## 写在前面

这是一个在慕课上面看到的一个课程练手的response

Go语言开发分布式任务调度 轻松搞定高性能Crontab 



开始日期，2019.4.1

结束日期，2019.4.15

整个跨度还是有点长的，半个月，两周，感觉还可以，对golang，etcd，crontab，mongodb有了一个初步的了解和使用。前面介绍新知识的时候看的很快，到后面重复性的东西比较多。



感觉整个项目就是在基于etcd来写，所有的核心功能都是用etcd来实现的，当然日志除外。



## 架构方面

这个我截了视频里面的一张图，争取用自己的话给讲出来

![1](C:\Users\Immortals\Desktop\1.png)



其实不难的，从上向下看。先说master。master主要分成两个部分，前端和后端。前端其实就是一个用于展示的web页面，静态的，没什么技术含量。ApiServer用于给前端提供接口。比如查看，新建，强杀等。配置文件主要是给后端提供一些基本的配置信息，比如读写超时，服务端口，集群列表等等。服务接口通过调用下面的小模块实现功能。

JobMgr 顾名思义，提供对job的管理，新建，删除，查看都是这个类实现的，具体实现方法就是依靠etcd，插入的话就是向 /cron/jobs/ 目录下存储key，然后worker节点对这个目录进行监听，监听到了有变化就拿出来，进行value的解析和执行。其他的功能都一样，删除也是写一个key进去，不过是在 /cron/killer/ 目录下，当然有另一个killer的后台负责监听这个目录的变化，然后就解析，执行。



WorkerMgr 实现服务的发现，就是查看能有几个还能继续工作的worker节点。实现方法也是基于etcd。简单来说，就是启动一个worker就向 /cron/workers/ 目录下注册一个自己的IP，master就通过对这个目录下所有keyt的读取来实现发现工作节点的。



LogMgr 日志服务，通过mongodb来实现日志。都是mongodb的东西，不难。



中间那两块核心，etcd和mongodb。简单说一下，etcd就是分布式key-value存储，mongodb，nosql数据库。东西网上都查的到，这里不多说。



然后是worker节点。master节点负责插入删除查看，那么worker节点就负责解析执行和日志。单从字面上来看，worker节点就要复杂一点。配置管理还是一些基础设置，日志大小，mongodb地址啥的。



Register 注册服务，目的是向etcd注册一个key，主要为了通知master我有一个节点上线，可以执行任务了。



JobMgr 负责监听 /cron/jobs/ 目录下的变化，里面有一个loop。执行流程是这样的，每当有一个新任务从master加入到etcd，那么这个类就会把这个新任务放入到jobPlanTable里面。loop不断调用trySchedule()，这个类的作用是从jobPlanTable里面遍历，找到时间到了的那个类，尝试执行。执行的时候调用Executor这个类，这个类先进行任务的抢锁（乐观锁），如果抢到锁，就执行，抢不到的话的等待下一轮。任务执行完成之后会通LogSink这个类存储到mongodb中去。



## 总结

这门课只是对这几个东西做了聚合整理而已，对底层没有涉及，所以对我这种初学者可能不太友好。就是知道这么做可以达成这个目的，但是不知道底层代码具体是怎么实现的，也不知道有没有其他办法来达成相同的目的。不过我的目的就止于此就可以了，了解了基本分布式的架构，了解了golang语言就可以了。现在实验室还没有要求，以后大概率是写C/C++，golang感觉学到这就可以了。如果以后有需要再继续。有缘再见。