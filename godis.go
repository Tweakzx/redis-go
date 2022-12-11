package godis

import (
	"log"
	"os"
)

func initServer(config *Config) Error {
	server.port = config.Port 
	server.clients = ListCreate(ListType{EqualFunc: ClientEqual})
	server.db = &Godis{
		data: DictCreate(DictType{HashFunc: GStrHash, EqualFunc: GStrEqual}),
		expire: DictCreate(DictType{HashFunc: GSteHash, EqualFunc: GStrEqual}), 
	}
	server.aeloop = AeLoopCreate()
	var err Error
	server.fd, err = TcpServer(server.port)
	
	return err
}

func main() {
	path := os.Args[1]
	config, err := LoadConfig(path)
	if err != nil {
		log.Printf("config error: %v \n", err)
	}
	err = initServer(config)
	if err != nil{
		log.Printf("init Server error: %v", err)
	}
	server.aeLoad,AddFileEvent(server, fd, AE_READABLE, AcceptHandler, nil)
	server.aeLoad.AddTimeEvent(AE_NORMAL, 1, ServerCron, nil)
	log.Print("godis server is up")
	server.aeLoop.AeMain()
}
