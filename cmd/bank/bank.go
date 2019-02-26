package main

import (
	"fmt"
	"os"

	"github.com/google/uuid"

	"github.com/screwyprof/cqrs"
	"github.com/screwyprof/cqrs/aggregatestore"
	"github.com/screwyprof/cqrs/commandhandler/bus"
	eventbus "github.com/screwyprof/cqrs/eventbus/memory"
	eventstore "github.com/screwyprof/cqrs/eventstore/memory"
	"github.com/screwyprof/cqrs/identitymap"
	"github.com/screwyprof/cqrs/middleware/commandhandler/transactional"
	"github.com/screwyprof/cqrs/repository"

	"github.com/screwyprof/cqrs/example/bank/command"
	"github.com/screwyprof/cqrs/example/bank/domain/account"
)

func main() {
	cqrs.RegisterAggregate(
		func(ID uuid.UUID) cqrs.ComplexAggregate {
			return account.Construct(ID)
		},
	)

	eventBus := eventbus.NewEventBus()
	identityMap := identitymap.NewIdentityMap()
	domainEventStorage := eventstore.NewEventStore()
	eventStoreUoW := aggregatestore.NewAggregateStore(domainEventStorage, identityMap, eventBus)
	domainRepository := repository.NewDomainRepository(identityMap, eventStoreUoW)

	commandBus := bus.NewCommandHandler(domainRepository)
	commandBus = cqrs.UseCommandHandlerMiddleware(commandBus, transactional.NewMiddleware(eventStoreUoW))

	accID := uuid.New()
	err := commandBus.Handle(command.OpenAccount{AggID: accID, Number: "ACC777"})
	failOnError(err)

	err = commandBus.Handle(command.DepositMoney{AggID: accID, Amount: 500})
	failOnError(err)
}

func failOnError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
