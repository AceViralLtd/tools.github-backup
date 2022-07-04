package worker

import (
	"sync"

	"github.com/google/go-github/v34/github"
)

var WaitGroup sync.WaitGroup
var glacierQueue chan QueueEntry
var ArciveQueue chan QueueEntry

type QueueEntry struct {
	Repo        *github.Repository
	Description string
}
