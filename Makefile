.PHONY: build deploy

VERSION?=$(shell git describe --tags --abbrev=0 HEAD)
GO_BUILD_CMD = CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -v -ldflags="-s -w" -ldflags="-X main.version=$(VERSION)" -trimpath
AWS_ACCOUNT_ID ?= $(shell aws sts get-caller-identity --query Account --output text)
AWS_REGION ?= $(shell aws configure get region)
BUCKET_NAME = auto-finance-deployment-$(AWS_ACCOUNT_ID)

create-deployment-bucket:
	aws s3api create-bucket --bucket $(BUCKET_NAME) --region $(AWS_REGION) --create-bucket-configuration LocationConstraint=$(AWS_REGION)

build:
	$(GO_BUILD_CMD) -o bin/bootstrap cmd/auto-finance/main.go
	mkdir -p ./bin/config
	@echo "Copying config file to bin directory..."
	cp ./config/config.toml ./bin/config/config.toml
	zip -j -9 ./bin/auto-finance.zip ./bin/bootstrap ./bin/config/config.toml
	sam build -t deployment/template.yaml

clean:
	rm -rf bin
	rm -rf .aws-sam

deploy: build
	sam deploy --template-file deployment/template.yaml --stack-name auto-finance --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM --s3-bucket $(BUCKET_NAME) --s3-prefix auto-finance --region $(AWS_REGION)