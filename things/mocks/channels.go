package mocks

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/mainflux/mainflux/things"
)

var _ things.ChannelRepository = (*channelRepositoryMock)(nil)

type channelRepositoryMock struct {
	mu       sync.Mutex
	counter  int
	channels map[string]things.Channel
	things   things.ThingRepository
}

// NewChannelRepository creates in-memory channel repository.
func NewChannelRepository(repo things.ThingRepository) things.ChannelRepository {
	return &channelRepositoryMock{
		channels: make(map[string]things.Channel),
		things:   repo,
	}
}

func (crm *channelRepositoryMock) Save(channel things.Channel) (string, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	crm.channels[key(channel.Owner, channel.ID)] = channel

	return channel.ID, nil
}

func (crm *channelRepositoryMock) Update(channel things.Channel) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	dbKey := key(channel.Owner, channel.ID)

	if _, ok := crm.channels[dbKey]; !ok {
		return things.ErrNotFound
	}

	crm.channels[dbKey] = channel
	return nil
}

func (crm *channelRepositoryMock) One(owner, id string) (things.Channel, error) {
	if c, ok := crm.channels[key(owner, id)]; ok {
		return c, nil
	}

	return things.Channel{}, things.ErrNotFound
}

func (crm *channelRepositoryMock) All(owner string, offset, limit int) []things.Channel {
	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	prefix := fmt.Sprintf("%s-", owner)
	channels := make([]things.Channel, 0)

	if offset < 0 || limit <= 0 {
		return channels
	}

	// Since IDs starts from 1, shift everything by one.
	first := fmt.Sprintf("%s%012d", startID, offset+1)
	last := fmt.Sprintf("%s%012d", startID, offset+limit+1)

	for k, v := range crm.channels {
		if strings.HasPrefix(k, prefix) && v.ID >= first && v.ID < last {
			channels = append(channels, v)
		}
	}

	sort.SliceStable(channels, func(i, j int) bool {
		return channels[i].ID < channels[j].ID
	})

	return channels
}

func (crm *channelRepositoryMock) Remove(owner, id string) error {
	delete(crm.channels, key(owner, id))
	return nil
}

func (crm *channelRepositoryMock) Connect(owner, chanID, thingID string) error {
	channel, err := crm.One(owner, chanID)
	if err != nil {
		return err
	}

	thing, err := crm.things.One(owner, thingID)
	if err != nil {
		return err
	}
	channel.Things = append(channel.Things, thing)
	return crm.Update(channel)
}

func (crm *channelRepositoryMock) Disconnect(owner, chanID, thingID string) error {
	channel, err := crm.One(owner, chanID)
	if err != nil {
		return err
	}

	for _, t := range channel.Things {
		if t.ID == thingID {
			connected := make([]things.Thing, len(channel.Things)-1)
			for _, thing := range channel.Things {
				if thing.ID != thingID {
					connected = append(connected, thing)
				}
			}

			channel.Things = connected
			return crm.Update(channel)
		}
	}

	return things.ErrNotFound
}

func (crm *channelRepositoryMock) HasThing(chanID, key string) (string, error) {
	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	suffix := fmt.Sprintf("-%s", chanID)

	for k, v := range crm.channels {
		if strings.HasSuffix(k, suffix) {
			for _, t := range v.Things {
				if t.Key == key {
					return t.ID, nil
				}
			}
			break
		}
	}

	return "", things.ErrNotFound
}
