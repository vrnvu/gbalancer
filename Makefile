.PHONY = help test run

.DEFAULT_GOAL = help

help:
	@echo "---------------HELP-----------------"
	@echo "To run the project in dev: make run"
	@echo "To test: make test"
	@echo "------------------------------------"

test:
	go test
	
run:
	go run main.go