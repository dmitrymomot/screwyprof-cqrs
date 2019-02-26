package aggregatestore

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/screwyprof/cqrs"
)

type AggregateStore struct {
	eventStore  cqrs.TransactionalEventStorage
	eventBus    cqrs.EventPublisher
	identityMap cqrs.IdentityMap

	eventProviders map[uuid.UUID]cqrs.EventProvider
}

func NewAggregateStore(
	eventStorage cqrs.TransactionalEventStorage, identityMap cqrs.IdentityMap, eventBus cqrs.EventPublisher) *AggregateStore {
	return &AggregateStore{
		eventStore:  eventStorage,
		eventBus:    eventBus,
		identityMap: identityMap,

		eventProviders: make(map[uuid.UUID]cqrs.EventProvider),
	}
}

func (s *AggregateStore) ByID(ID uuid.UUID, aggregateType string) (cqrs.ComplexAggregate, error) {
	events, err := s.eventStore.LoadEventStream(ID)
	if err != nil {
		return nil, err
	}
	// maybe return eventstore.AggregateNotFoundError{ID:ID} if no events loaded

	aggregateRoot := cqrs.CreateAggregate(aggregateType, ID)
	if err := aggregateRoot.LoadFromHistory(events); err != nil {
		return nil, err
	}

	s.registerForTracking(aggregateRoot)

	return aggregateRoot, nil
}

func (s *AggregateStore) Add(aggregateRoot cqrs.ComplexAggregate) {
	s.registerForTracking(aggregateRoot)
}

func (s *AggregateStore) registerForTracking(aggregateRoot cqrs.ComplexAggregate) {
	s.eventProviders[aggregateRoot.AggregateID()] = aggregateRoot
	s.identityMap.Add(aggregateRoot)
}

func (s *AggregateStore) Commit() error {
	fmt.Println("AggregateStore: Commit")

	s.eventStore.BeginTransaction()

	var eventsToPublish []cqrs.DomainEvent
	for _, eventProvider := range s.eventProviders {
		err := s.eventStore.Store(
			eventProvider.AggregateID(), eventProvider.Version(), eventProvider.UncommittedChanges())
		if err != nil {
			return err
		}

		eventsToPublish = append(eventsToPublish, eventProvider.UncommittedChanges()...)
		//s.eventBus.Publish(eventProvider.UncommittedChanges()...)
		eventProvider.MarkChangesAsCommitted()
	}
	s.eventProviders = make(map[uuid.UUID]cqrs.EventProvider)

	//s.eventBus.Commit()
	if err := s.eventStore.Commit(); err != nil {
		return err
	}

	s.eventBus.Publish(eventsToPublish...)

	return nil
}

/*
func (s *AggregateStore) Commit() error {
	fmt.Println("AggregateStore: Commit")
	s.eventStore.BeginTransaction()

	for _, eventProvider  := range s.eventProviders {
		err := s.eventStore.Store(
			eventProvider.AggregateID(), eventProvider.Version(), eventProvider.UncommittedChanges())
		if err != nil {
			return err
		}

		s.eventBus.Publish(eventProvider.UncommittedChanges()...)
		eventProvider.MarkChangesAsCommitted()
	}
	s.eventProviders = make(map[uuid.UUID]cqrs.EventProvider)

	//s.eventBus.Commit()
	return s.eventStore.Commit()
}
*/

func (s *AggregateStore) Rollback() error {
	fmt.Println("AggregateStore: Rollback")
	//_bus.Rollback();
	err := s.eventStore.Rollback()
	if err != nil {
		return err
	}

	for _, eventProvider := range s.eventProviders {
		s.identityMap.Remove(eventProvider.AggregateID())
	}

	s.eventProviders = make(map[uuid.UUID]cqrs.EventProvider)
	return nil
}