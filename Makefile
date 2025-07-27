.PHONY: build deploy

VERSION?=local
GO_BUILD_CMD = CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -v -ldflags="-s -w" -ldflags="-X main.version=$(VERSION)" -trimpath
AWS_ACCOUNT_ID ?= $(shell aws sts get-caller-identity --query Account --output text)
AWS_REGION ?= $(shell aws configure get region)
BUCKET_NAME = auto-finance-deployment-$(AWS_ACCOUNT_ID)

create-deployment-bucket:
	aws s3api create-bucket --bucket $(BUCKET_NAME) --region $(AWS_REGION) --create-bucket-configuration LocationConstraint=$(AWS_REGION)

build:
	$(GO_BUILD_CMD) -o bin/bootstrap cmd/auto-finance/main.go
	cp ./config/config.toml bin/config.toml
	zip -j -9 bin/auto-finance.zip bin/bootstrap bin/config.toml
	sam build -t deployment/template.yaml


deploy: build
	sam deploy --template-file deployment/template.yaml --stack-name auto-finance --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM --s3-bucket $(BUCKET_NAME) --s3-prefix auto-finance --region $(AWS_REGION)