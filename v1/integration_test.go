package machinery

import (
	"fmt"
	"os"
	"testing"

	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/errors"
	"github.com/RichardKnop/machinery/v1/signatures"
)

func TestIntegration(t *testing.T) {
	brokerURL := os.Getenv("AMQP_URL")
	if brokerURL == "" {
		return
	}

	server1 := setup(brokerURL, "amqp")
	worker1 := server1.NewWorker("test_worker")
	go worker1.Launch()
	_testSendTask(server1, t)
	_testSendChain(server1, t)
	worker1.Quit()

	memcacheURL := os.Getenv("MEMCACHE_URL")
	if memcacheURL == "" {
		return
	}

	server2 := setup(brokerURL, fmt.Sprintf("memcache://%v", memcacheURL))
	worker2 := server2.NewWorker("test_worker")
	go worker2.Launch()
	_testSendTask(server2, t)
	_testSendChain(server2, t)
	worker2.Quit()
}

func _testSendTask(server *Server, t *testing.T) {
	task := signatures.TaskSignature{
		Name: "add",
		Args: []signatures.TaskArg{
			signatures.TaskArg{
				Type:  "int64",
				Value: 1,
			},
			signatures.TaskArg{
				Type:  "int64",
				Value: 1,
			},
		},
	}

	asyncResult, err := server.SendTask(&task)
	if err != nil {
		t.Error(err)
	}

	result, err := asyncResult.Get()
	if err != nil {
		t.Error(err)
	}

	if result.Interface() != int64(2) {
		t.Errorf(
			"result = %v(%v), want int64(2)",
			result.Type().String(),
			result.Interface(),
		)
	}
}

func _testSendChain(server *Server, t *testing.T) {
	task1 := signatures.TaskSignature{
		Name: "add",
		Args: []signatures.TaskArg{
			signatures.TaskArg{
				Type:  "int64",
				Value: 1,
			},
			signatures.TaskArg{
				Type:  "int64",
				Value: 1,
			},
		},
	}

	task2 := signatures.TaskSignature{
		Name: "add",
		Args: []signatures.TaskArg{
			signatures.TaskArg{
				Type:  "int64",
				Value: 5,
			},
			signatures.TaskArg{
				Type:  "int64",
				Value: 6,
			},
		},
	}

	task3 := signatures.TaskSignature{
		Name: "multiply",
		Args: []signatures.TaskArg{
			signatures.TaskArg{
				Type:  "int64",
				Value: 4,
			},
		},
	}

	chain := NewChain(&task1, &task2, &task3)
	chainAsyncResult, err := server.SendChain(chain)
	if err != nil {
		t.Error(err)
	}

	result, err := chainAsyncResult.Get()
	if err != nil {
		t.Error(err)
	}

	if result.Interface() != int64(52) {
		t.Errorf(
			"result = %v(%v), want int64(52)",
			result.Type().String(),
			result.Interface(),
		)
	}
}

func setup(brokerURL, backend string) *Server {
	cnf := config.Config{
		Broker:        brokerURL,
		ResultBackend: backend,
		Exchange:      "test_exchange",
		ExchangeType:  "direct",
		DefaultQueue:  "test_queue",
		BindingKey:    "test_task",
	}

	server, err := NewServer(&cnf)
	errors.Fail(err, "Could not initialize server")

	tasks := map[string]interface{}{
		"add": func(args ...int64) (int64, error) {
			sum := int64(0)
			for _, arg := range args {
				sum += arg
			}
			return sum, nil
		},
		"multiply": func(args ...int64) (int64, error) {
			sum := int64(1)
			for _, arg := range args {
				sum *= arg
			}
			return sum, nil
		},
	}
	server.RegisterTasks(tasks)

	return server
}
