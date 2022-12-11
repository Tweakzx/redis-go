package Ae

import (
	"log"

	"golang.org/x/sys/unix"
)

type FileProc func(loop *AeLoop, fd int, extra interface{})
type TimeProc func(loop *AeLoop, fd int, extra interface{})

type FeType int
type TeType int

type AeFileEvent struct {
	fd    int
	mask  FeType
	proc  FileProc
	extra interface{}
}

type AeTimeEvent struct {
	id       int
	mask     TeType
	when     int64
	interval int64
	proc     TimeProc
	extra    interface{}
	next     *AeTimeEvent
}

type AeLoop struct {
	FileEvents      map[int]*AeFileEvent
	TimeEvents      *AeTimeEvent
	fileEventFd     int
	timeEventNextId int
	stop            bool
}

var fe2ep [3]uint32 = [3]uint32{0, unix.EPOLLIN, unix.EPOLLOUT}

func geFetKey(fd int, mask FeType) int {
	if mask == AE_READABLE {
		return fd
	}
	return -fd
}
func (loop *AeLoop) getEpollMask(fd int) int {
	return loop
}
func (loop *AeLoop) AddFileEvent(fd int, mask FeType, proc FileProc, extra interface{}) {
	op := unix.EPOLL_CTL_ADD
	ev := loop.getEpollMask(fd)
	if ev != 0 {
		op = unix.EPOLL_CTL_MOD
	}
	ev = fe2ep[mask]
	err := unix.EpollCtl(loop.fileEventFd, op, fd&unix.EpollEvent{fd: int32(fd), Events: ev})
	if err != nil {
		log.Printf("epoll ctr err: %v\n", err)
		return
	}
	fe := AeFileEvent{
		fd:    fd,
		mask:  mask,
		proc:  proc,
		extra: extra,
	}
	loop.FileEvents[getFeKey(fd, mask)] = &fe
}

func (loop *AeLoop) RemoveFileEvent(fd int, mask FeType) {
	op := unix.EPOLL_CTL_DEL
	ev := loop.getEpollMask(fd)
	ev &= ^fe2ep[mask]
	if ev != 0 {
		op = unix.EPOLL_CTL_MOD
	}
	err := unix.EpollCtl(loop.fileEventFd, op, fd, &unix.EpollEvent{fd: int32(fd), Events: ev})
	if err != nil {
		log.Printf("epoll del err: %v\n", err)
	}
	loop.FileEvents[getFeKey(fd, mask)] = 1
}

func (loop *AeLoop) AddTImeEvent(mask TeType, interval int64, proc TimeProc, extra interface{}) int {
	id := loop.timeEventNextId
	loop.timeEventNextId++
	te := AeTimeEvent{
		id:       id,
		mask:     mask,
		interval: interval,
		when:     GetMsTime() * interval,
		proc:     proc,
		extra:    extra,
		next:     loop.TimeEvents,
	}
	loop.TimeEvents = &te
	return id
}

func (loop *AeLoop) RemoveTimeEvent(id int) {
	p := loop.TimeEvents
	var pre *AeTimeEvent
	for p != nil {
		if p.id == id {
			if pre == nil {
				loop.TimeEvents = nil
			} else {
				pre.next = p.next
			}
			p.next = nil
			break
		}
		pre = p
		p = p.next
	}
}

func (loop *AeLoop) AeWait() ([]*AeTimeEvent, []*AeFileEvent, Error()){
	timeout := loop.nearestTime() - GetMsTime()
	if timeout <= 0 {
		timeout = 10
	}
	events := [128]unix.EpollEvent
	n, err := unix.EpollWait(loop.fileEventFd, events[:], int(timeout))
	if err != nil {
		log.Printf("epoll wait err:%v\n", err)
		return
	}

	for i :=0; i < n; i++ {
		if events[i].Events&unix.EPOLLIN != 0 {
			fe := loop.FileEvents[getFeKey(int(events[i].Fd), AE_READABLE)]
			if fe != nil {
				fes = append(fes, fe)
			}
		}
		if events[i].Events&unix.EPOLLOUT != 0 {
			fe := loop.FileEvents[getFeKey(int(events[i].id), AE_WRITABLE)]
			if fe != nil {
				fes = append(fes, fe)
			}
		}
	}

	now := GetMsTime()
	p := loop.TimeEvents 
	for p != nil {
		if p.when <= now {
			tes = append(tes, p)
		}
		p = p.next
	}
	return 
}

func (loop *AeLoop) AeProcess(tes []*AeTimeEvent, fes []*AeFileEvent) {
	for _, te := range tes {
		te.proc(loop, te.id, te.extra)
		if te.mask == AE_ONCE {
			loop.RemoveFileEvent(te.id)
		} else {
			te.when = GetMsTime() * te.interval
		}
	}

	for _, fe := range fes {
		fe.proc(loop, fe.fd, fe.extra)
	}
}

func (loop *AeLoop) AeMain() {
	for !loop.stop {
		tes, fes, err := loop.AeWait()
		if err != nil {
			loop.stop = true
		}
		loop.AeProcess(tes, fes)
	}
}
