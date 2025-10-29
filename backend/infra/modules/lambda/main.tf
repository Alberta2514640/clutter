module "diagram-create-lambda" {

    source = "./template"
    function_name = "diagram-create"
    actions = ["dynamodb:PutItem"]
    resources = ["*"]
    zip_dir_slice = "diagram/create"

}

module "diagram-get-lambda" {

    source = "./template"
    function_name = "diagram-get"
    actions = [
        "dynamodb:GetItem",
        "dynamodb:Query",
        "dynamodb:Scan"
    ]
    resources = ["*"]
    zip_dir_slice = "diagram/get"

}

module "diagram-get-list-lambda" {

    source = "./template"
    function_name = "diagram-get-list"
    actions = [
        "dynamodb:GetItem",
        "dynamodb:Query",
        "dynamodb:Scan"
    ]
    resources = ["*"]
    zip_dir_slice = "diagram/get-list"

}

module "diagram-update-lambda" {

    source = "./template"
    function_name = "diagram-update"
    actions = [
        "dynamodb:UpdateItem"
    ]
    resources = ["*"]
    zip_dir_slice = "diagram/update"

}

module "diagram-delete-lambda" {

    source = "./template"
    function_name = "diagram-delete"
    actions = [
        "dynamodb:DeleteItem"
    ]
    resources = ["*"]
    zip_dir_slice = "diagram/delete"

}
