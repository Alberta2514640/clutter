# Introduction

In this brief tutorial, you will learn how to deploy and modify the infrastructure for Clutter via Terraform.

## What are you looking at?

The first thing to understand is what you are working with. In a nutshell, we have code, in this case Go code, that we want to deploy to AWS as a Lambda function. The code for this currently lives in the `clutter/backend/api` folder. Taking a quick look at this folder you will notice some folders and files.
- `go.mod` is a module file which helps Go (and other developers) keep track of what packages your project depends on, the Go version you are using and the name of your module.
- `go.sum` is a file that essentially performs a security check to make sure your project’s dependencies haven’t changed or been messed with (not too important to worry about as the developer)
- `/generic` is a folder I have created to store generic functions we will be using accross our different Lambdas. Currently is has a file called `responses.go` and this contains a generic `Response()` function. This function is standard for all of our Lambdas and so to reduce repeated code, we define it here and use it wherever we need it. You can use this as an example to create other generic functions.
- `/diagram` is a folder that will house each CRUD Lambda function related to the diagram (create-diagram, delete-diagram, etc.). Others will be created just like it for organization, project, and more that we may need down the line.
- `/diagram/create` is the folder which has the source code for its associated lambda function (create-diagram). Its source code is found in `api/diagram/create/main.go`, and this is the file that gets built into a binary, zipped and uploaded to AWS Lambda. Similarly, a folder for each of the CRUD operations would be created to create a tree that would look like the following:
<pre>
backend/
└── api/
    ├── diagram/
    │   ├── create/
    │   ├── get/
    │   ├── get-list/
    │   ├── update/
    │   └── delete/
    └── project/
        ├── create/
        ├── get/
        ├── get-list/
        ├── update/
        └── delete/
</pre>
- `build.sh` is a Bash script which will allow us to quickly build each of our `main.go` Lambda functions into binaries and zip them to be deployed to AWS (the resulting files are sent to `/deploy/` found in each Lambda's respective folder). This script finds a `main.go` in a list of Lambda directories we define (this needs to manually be updated for every new Lambda function we make), builds it into an executable binary (`bootstrap`) and zips it (`bootstrap.zip`). It makes this binary for a Linux OS and for an `arm64` architecture since we are deploying these Lambdas to arm64 Amazon Linux 2 machines.

Now that we have covered what the Go code structure looks like, I would like to quickly cover the `backend/infra` folder (ignore `backend/tfstate-infra`, however if you would like to learn why this exists feel free to ask me).
- `/modules` contains IaC for the different AWS services we are going to deploy, organized into their own folders and further seperated within these folders by context (ex. in `/lambda` there will be IaC for each different function in its own folder as `lambda/create-diagram`)
- `backend.tf` defines the remote storage for the terraform state and lock files (these files save the state of the infrastructure and lock updating to one person at a time)
- `main.tf` contains the modules to deploy from the `/modules` folder
- `providers.tf` defines the AWS version and other information regarding region
- `terraform.tfvars` is a file where you can define variables to be used in the `main.tf` files
- `variables.tf` lets the `main.tf` know which variables are defined in `terraform.tfvars`. Without this file, `main.tf` will not know what variables are stored in `terraform.tfvars`

## Pre-requisites (in no particular order)

1. Install the latest Go version (1.25.3 as of writing this) from https://go.dev/doc/install
   - check by running `go version` in terminal
2. Install Terraform CLI from https://developer.hashicorp.com/terraform/tutorials/aws-get-started/install-cli
   - check by running `terraform -v` in terminal
4. Install the official Go VS Code extension (https://marketplace.visualstudio.com/items?itemName=golang.Go)
5. `git pull` this branch (Maitra-application-infra)
6. run `git checkout Maitra-application-infra`

## Steps for Deployment
1. `cd backend/api`
2. `./build.sh`
   - this will build all the `main.go` files into their respective `/deploy` folders
3. `cd ../infra`
4. Navigate to [AWS Access Portal](https://d-9267ff1cd3.awsapps.com/start/) and log in
5. Under `AWS Accounts` select the `Capstone` dropdown
6. Select `Access Keys`
7. Select the terminal you are using (MacOS, Windows, Powershell)
8. Copy the text for `Option 1: Set AWS environment variables`
9. Paste into your terminal and confirm the environment variables have been stored correctly by printing them out
   - `echo $AWS_ACCESS_KEY_ID` as one example
10. Navigate to the `Capstone` Discord server, then to the `Miscellaneous>important-links` channel where you will find `Terraform tfvars file needed to start deploying`. Download this file and paste it into the `/backend/infra` folder
11. `terraform init`
12. `terraform apply`
    - you will be prompted to perform actions. You can type `yes` to continue
13. If the output prints `Apply complete! Resources: 0 added, 1 changed, 0 destroyed.` you have successfully deployed AWS resources using Terraform!
    - the reason it says `1 changed` is because you zipped the source code on your machine and Terraform keeps track of any changes to the "code" by using a source_code_hash comparison. If it detects a new hash, it will assume a change has been made, and since you ran `.build.sh` it is either creating a new zip file or overwriting the old zip file. This doesn't always mean something in your code or the functionality changed, just that you made a new zip file.
