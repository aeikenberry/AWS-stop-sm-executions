package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sfn"
)

type stepStuff struct {
	Client     sfn.SFN
	Executions chan string
	Done       chan bool
}

func getAllStateMachineArns(s *stepStuff) []*sfn.StateMachineListItem {
	resp, err := s.Client.ListStateMachines(&sfn.ListStateMachinesInput{})
	if err != nil {
		log.Fatal(err.Error())
	}
	return resp.StateMachines
}

func getMachineExecutions(s *stepStuff, arn string) {
	resp, err := s.Client.ListExecutions(&sfn.ListExecutionsInput{
		StateMachineArn: aws.String(arn),
		StatusFilter:    aws.String("RUNNING"),
	})
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, ex := range resp.Executions {
		s.Executions <- *ex.ExecutionArn
	}
}

func getAllExecutions(s *stepStuff) {
	machines := getAllStateMachineArns(s)

	for _, machine := range machines {
		getMachineExecutions(s, *machine.StateMachineArn)
	}

	close(s.Executions)
}

func cancelExecutions(s *stepStuff) {
	var count int

	for {
		e, more := <-s.Executions

		if more {
			_, err := s.Client.StopExecution(&sfn.StopExecutionInput{
				ExecutionArn: aws.String(e),
			})
			if err != nil {
				log.Fatal(err.Error())
			}
			count = count + 1
		} else {
			fmt.Printf("Cancelled %d executions. \n", count)
			s.Done <- true
			return
		}
	}
}

func cancelAll(step *stepStuff) {
	go cancelExecutions(step)
	getAllExecutions(step)
	<-step.Done
}

func main() {
	var region string

	flag.StringVar(&region, "region", "us-east-1", "AWS Region")

	sess := session.Must(session.NewSession())
	stepClient := sfn.New(sess, &aws.Config{Region: aws.String(region)})
	executions := make(chan string, 4)
	done := make(chan bool)

	cancelAll(&stepStuff{
		Client:     *stepClient,
		Executions: executions,
		Done:       done,
	})
}
