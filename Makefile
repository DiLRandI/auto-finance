.PHONY: build deploy copy-config

VERSION?=$(shell git describe --tags --abbrev=0 HEAD)
GO_BUILD_CMD = CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -v -ldflags="-s -w" -ldflags="-X main.version=$(VERSION)" -trimpath
AWS_ACCOUNT_ID ?= $(shell aws sts get-caller-identity --query Account --output text)
AWS_REGION ?= $(shell aws configure get region)
BUCKET_NAME = auto-finance-deployment-$(AWS_ACCOUNT_ID)
CONFIG_BUCKET = $(shell aws cloudformation describe-stacks --stack-name auto-finance --query "Stacks[0].Outputs[?OutputKey=='ConfigurationBucketName'].OutputValue" --output text)
ENV?=dev

create-deployment-bucket:
	aws s3api create-bucket --bucket $(BUCKET_NAME) --region $(AWS_REGION) --create-bucket-configuration LocationConstraint=$(AWS_REGION)

build:
	$(GO_BUILD_CMD) -o bin/bootstrap cmd/auto-finance/main.go
	mkdir -p ./bin/config
	zip -j -9 ./bin/auto-finance.zip ./bin/bootstrap
	sam build -t deployment/template.yaml

clean:
	rm -rf bin
	rm -rf .aws-sam

deploy: build copy-config
	sam deploy --template-file deployment/template.yaml --stack-name auto-finance-$(ENV) \
		--capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM --s3-bucket $(BUCKET_NAME) \
		--s3-prefix auto-finance-$(ENV) --region $(AWS_REGION) \
		--parameter-overrides ENV=$(ENV) \
		--tags "AppManagerCFNStackKey=auto-finance-$(ENV) AppManagerCFNStackName=auto-finance-$(ENV) Application=auto-finance-$(ENV) Environment=$(ENV)" 

info:
	@echo "AWS Account ID: $(AWS_ACCOUNT_ID)"
	@echo "AWS Region: $(AWS_REGION)"
	@echo "Deployment Bucket: $(BUCKET_NAME)"
	@echo "Config Bucket: $(CONFIG_BUCKET)"
	@echo "Version: $(VERSION)"
	@echo "Build Command: $(GO_BUILD_CMD)"
	@echo "SAM Build Command: sam build -t deployment/template.yaml"

copy-config:
	@echo "Copying config file to S3..."
# 	aws s3 cp ./config.toml s3://$(CONFIG_BUCKET)/config.toml
