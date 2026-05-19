.PHONY: install run

install:
	go install .

run:
	go run . $(ARGS)
