.PHONY: genmock
genmock:
	@echo "Generating mocks..."
	@go run go.uber.org/mock/mockgen@latest \
		-destination=cachex_mock_test.go \
		-package=$(shell basename $$(pwd)) \
		-source=cachex.go \
		Cacher
	@echo "Mocks generated successfully!!!"