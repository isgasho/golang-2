package worker

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"golang/crontab/common"
	"time"
)

// 任务管理器
type JobMgr struct{
	client *clientv3.Client
	kv clientv3.KV
	lease clientv3.Lease
	watcher clientv3.Watcher
}

var(
	// 单例
	G_jobMgr *JobMgr
)

// 监听任务变化
func (jobMgr *JobMgr)watchjobs()(err error)  {
	var(
		getResp *clientv3.GetResponse
		kvpair *mvccpb.KeyValue
		job *common.Job
		watchStartRevision int64
		watchChan clientv3.WatchChan
		wacthResp clientv3.WatchResponse
		watchEvent *clientv3.Event
		jobName string
		jobEvent *common.JobEvent
	)
	// 1.get一下/cron/jobs/目录所有文件，并且获知当前集群的revision
	if getResp,err = jobMgr.kv.Get(context.TODO(),common.JOB_SAVE_DIR,clientv3.WithPrefix());err != nil{
		return
	}

	// 当前有哪些任务
	for _,kvpair = range getResp.Kvs{
		// 反序列化json得到job
		if job,err = common.UnpackJob(kvpair.Value);err == nil{
			// 把这个job同步给schedule（调度协程）
			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE,job)

			G_scheduler.PushJobEvent(jobEvent)
		}
	}

	// 从该revision向后监听事件变化
	go func() {
		// 监听协程
		// 从GET时刻的后续版本开始监听变化
		watchStartRevision = getResp.Header.Revision + 1
		// 监听/cron/jobs/目录的后续变化
		watchChan = jobMgr.watcher.Watch(context.TODO(),common.JOB_SAVE_DIR,clientv3.WithRev(watchStartRevision),clientv3.WithPrefix())
		// 处理监听事件
		for wacthResp = range watchChan{
			for _,watchEvent = range wacthResp.Events{
				switch watchEvent.Type {
				case mvccpb.PUT: // 任务保存事件
					if job,err = common.UnpackJob(watchEvent.Kv.Value);err != nil{
						continue
					}
					// 构造一个更新Event事件
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE,job)
				case mvccpb.DELETE: // 任务删除事件
					// Delete /cron/jobs/job10
					jobName = common.ExtractJobName(string(watchEvent.Kv.Key))
					job = &common.Job{Name:jobName}
					// 构建一个删除Event
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE,job)
				}
				// 推事件给scheduler
				G_scheduler.PushJobEvent(jobEvent)
			}
		}
	}()

	return
}

func InitJobMgr() (err error) {
	var(
		config clientv3.Config
		client *clientv3.Client
		kv clientv3.KV
		lease clientv3.Lease
		watcher clientv3.Watcher
	)

	// 初始化配置
	config = clientv3.Config{
		Endpoints: G_config.EtcdEndpoints,	// 集群地址
		DialTimeout:time.Duration(G_config.EtcdDialTimeout)*time.Millisecond,	// 链接超时
	}

	// 建立连接
	if client,err = clientv3.New(config);err != nil{
		return
	}

	// 得到KV和Lease的api子集
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	watcher = clientv3.NewWatcher(client)

	// 赋值单例
	G_jobMgr = &JobMgr{
		client:client,
		kv:kv,
		lease:lease,
		watcher:watcher,
	}

	// 启动任务监听
	G_jobMgr.watchjobs()

	return
}