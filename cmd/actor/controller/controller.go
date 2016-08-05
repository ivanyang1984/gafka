package controller

import (
	"sync"

	"github.com/funkygao/gafka/zk"
	log "github.com/funkygao/log4go"
)

type Controller interface {
	ServeForever() error
	Stop()
}

type controller struct {
	orchestrator *zk.Orchestrator
	wg           sync.WaitGroup
	quiting      chan struct{}
}

func New(zkzone *zk.ZkZone) Controller {
	return &controller{
		quiting:      make(chan struct{}),
		orchestrator: zkzone.NewOrchestrator(),
	}
}

func (this *controller) ServeForever() (err error) {
	id := this.id()
	if err = this.orchestrator.RegisterActor(id); err != nil {
		return err
	}

	for {
		// each loop is a new rebalance process

		select {
		case <-this.quiting:
			return nil
		default:
		}

		jobs, jobChanges, err := this.orchestrator.WatchJobQueues()
		if err != nil {
			return err
		}

		actors, actorChanges, err := this.orchestrator.WatchActors()
		if err != nil {
			return err
		}

		decision := assignJobsToActors(actors, jobs)
		myJobs := decision[id]

		if len(myJobs) == 0 {
			// standby mode
			log.Warn("no job assignment, awaiting rebalance...")
		}

		workStopper := make(chan struct{})
		for _, job := range myJobs {
			this.wg.Add(1)
			go this.workOnJob(job, workStopper)
		}

		select {
		case <-this.quiting:
			close(workStopper)
			//return

		case <-jobChanges:
			close(workStopper)
			this.wg.Wait()

		case <-actorChanges:
			stillAlive, err := this.orchestrator.ActorRegistered(id)
			if err != nil {
				log.Error(err)
			} else if !stillAlive {
				this.orchestrator.RegisterActor(id)
			}

			close(workStopper)
			this.wg.Wait()
		}

	}

}

func (this *controller) Stop() {
	close(this.quiting)
}

func (this *controller) id() string {
	return ""
}

func (this *controller) workOnJob(job string, stopper <-chan struct{}) {
	defer this.wg.Done()

}