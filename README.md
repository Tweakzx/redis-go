# godis

## 深入理解redis

### Redis 源码结构

redis1.0
|                   //主体
|-redis.c               //逻辑
|-redis-cli.c           //客户端
|                   //库
|-ae.c                  //事件库
|-ae.net                //网络库
|-zmalloc.c             //内存库
|                   //数据结构
|-sds.c                 //简单动态字符串
|-adlist.c              //双向链表
|-dict.c                //字典
|                   //辅助
|-qpsort.c              //排序
|-benchmark.c           //benchmark
|-lzf.c                 //压缩算法

### 核心概念

- Redis-server
  - fd            文件句柄
  - db            *RedisDBList
  - client        *RedisClientList
  - eventLoop     *AeEventLoop
- Redis Client
  - fd            int
  - db            *RedisDB
  - query         string
  - reply         *RedisObjList
- RedisDB
  - dict          *Dict
  - expires       *Dict
  - id            int
- RedisObj
  - ptr           *void 指向任意类型的指针
  - type          int类型
  - refCount      int 引用计数做内存管理， 计数为0的时候释放

### 核心流程

- 启动流程

```c
int main(int argc, char **argv) {
    //初始化config
    initServerConfig(); 
    ...
    //初始化Server
    initServer();   
    ...
    //rdb的处理
    if (rdbLoad(server.dbfilename) == REDIS_OK)
        redisLog(REDIS_NOTICE,"DB loaded from disk");
    //针对server.fd 注册了一个FileEvent
    if (aeCreateFileEvent(server.el, server.fd, AE_READABLE,
        acceptHandler, NULL, NULL) == AE_ERR) oom("creating file event");
    redisLog(REDIS_NOTICE,"The server is now ready to accept connections on port %d", server.port);
    //ae的事件循环
    aeMain(server.el);
    aeDeleteEventLoop(server.el);
    return 0;
}

```

```c
static void initServer() {
    ...
    // 创建
    server.clients = listCreate();
    server.slaves = listCreate();
    server.monitors = listCreate();
    server.objfreelist = listCreate();
    createSharedObjects();
    server.el = aeCreateEventLoop();
    server.db = zmalloc(sizeof(redisDb)*server.dbnum);
    server.sharingpool = dictCreate(&setDictType,NULL);
    ...
    //启动TCP
    server.fd = anetTcpServer(server.neterr, server.port, server.bindaddr);
    ...
    //初始化db
    for (j = 0; j < server.dbnum; j++) {
        server.db[j].dict = dictCreate(&hashDictType,NULL);
        server.db[j].expires = dictCreate(&setDictType,NULL);
        server.db[j].id = j;
    }
    ...
    //初始化event
    aeCreateTimeEvent(server.el, 1000, serverCron, NULL, NULL);
}
```

- AE库

  - IO多路复用：IO就绪之前，主线程先去做别的事，等到完全就绪之后回调主线程处理数据
  - IO原理
  - 两类事件
    - 文件事件
    - 时间事件
  - 方法调用流程
    - aeCreateFileEvent(...)：头插法创造并插入事件
    - aeMain(...)：不断处理各种Event
    - acceptHanlde(...)
    - crearteClient(...)
    - readQueryClient(...)
    - processCommand(...)
    - getCommand(...):在table中找到
  - 请求处理流程
    - request
    - handle
    - reply
  - ![redis-eventloop-proces-event](https://img.draveness.me/2016-12-09-redis-eventloop-proces-event.png-1000width)

### 核心数据结构

- Dict KV数据库

  - 结构

    ```
    DB
     |-->Expire
     |-->Data-->hashtable-->table
    		          |-->(entry-->entry)
      		          |-->(entry-->entry)
    ```
  - expire: 单线程，扫描成本太大， 不能扫描删除过期data， 只能用这个Key的时候check是否过期，过期则删掉值，返回空（lazy delete）：双字典+lazydelete
  - rehash: 渐进哈希，扩容，rehash
- RedisObj

  - 机制：通过refCount进行内存管理
  - set生命周期：
    ```
    set (k, v)-->processCOmmand()-->setCommand-->reply "OK"
    		|
    		|-->createObject-->redisClient-->argv[key, val]-->freeClientargv
    ```
  - get生命周期
    ```
    get(k) --> processCommand-->getCommand-->sendReplyToCLient		 <--|
    		|		|					    |
    		|		|->dictFind()resObj-->addRelpy()-->reply-->val
    		|
    		|->createObject-->redisClient-->argv[key, val]-->freeClientargv
    ```
- adlist
- sds
