package deaddrop

import (
	"os"
	"io"
	"log"
	"time"
	"sync"
)

type Collection struct {
	drops map[string]*deadDrop
	mutex sync.Mutex
}

type deadDrop struct {
    mutex sync.Mutex
	key              string
	collection       *Collection
	source           chan io.Reader
	target           chan io.Writer
	sourceCompletion chan os.Error
	targetCompletion chan os.Error
}

func New() *Collection {
	return &Collection{drops: make(map[string] *deadDrop)}
}

func (collection *Collection) SetSource(source io.Reader, key string) (completion chan os.Error, ok bool) {

	drop := collection.dropForKey(key)

	drop.mutex.Lock()
	defer drop.mutex.Unlock()

	if drop.sourceCompletion == nil {
		completion = make(chan os.Error)
		drop.sourceCompletion = completion
		drop.source <- source
		ok = true
	} else {
		ok = false
	}
	return
}

func (collection *Collection) SetTarget(target io.Writer, key string) (completion chan os.Error, ok bool) {
	drop := collection.dropForKey(key)

	drop.mutex.Lock()
	defer drop.mutex.Unlock()

	if drop.targetCompletion == nil {
		completion = make(chan os.Error)
		drop.targetCompletion = completion
		drop.target <- target
		ok = true
	} else {
		ok = false
	}
	return
}

func (collection *Collection) dropForKey(key string) *deadDrop {
	collection.mutex.Lock()
	defer collection.mutex.Unlock()

	drop, present := collection.drops[key]
	if !present {
		log.Print("Create new drop for key: ", key)
		drop = &deadDrop{}
		collection.drops[key] = drop
		drop.source = make(chan io.Reader)
		drop.target = make(chan io.Writer)

		go func() {
			timeout := make(chan bool)
			timer := time.AfterFunc(60*1000000000, func() {
				timeout <- true
			})

			var source io.Reader
			var target io.Writer
			for source == nil || target == nil {
				select {
				case source = <-drop.source:
					log.Printf("[%s] Set source.", key)
				case target = <-drop.target:
					log.Printf("[%s] Set target.", key)
				case <-timeout:
					log.Printf("[%s] Timeout", key)
					if drop.sourceCompletion != nil {
						drop.sourceCompletion <- os.NewError("Tiemout")
					}
					if drop.targetCompletion != nil {
						drop.targetCompletion <- os.NewError("Timeout")
					}
					return
				}
			}
			timer.Stop()

			log.Printf("[%s] Remove drop from collection.", key)
			collection.mutex.Lock()
			collection.drops[key] = nil, false
			collection.mutex.Unlock()

			log.Printf("[%s] Start copying ...", key)

			n, err := io.Copy(target, source)
			if err != nil {
				log.Printf("[%s] Faild copying with error: %s", key, err.String())
			} else {
				log.Printf("[%s] Copied %d bytes.", key, n)
			}

			if drop.sourceCompletion != nil {
				drop.sourceCompletion <- err
			}

			if drop.targetCompletion != nil {
				drop.targetCompletion <- err
			}
		}()
	}
	return drop
}

