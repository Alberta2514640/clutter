# diagram-create
output "diagram_create_function_name" {
    value = module.diagram-create-lambda.function_name
}
output "diagram_create_invoke_arn" {
    value = module.diagram-create-lambda.invoke_arn
}

# diagram-get
output "diagram_get_function_name" {
    value = module.diagram-get-lambda.function_name
}
output "diagram_get_invoke_arn" {
    value = module.diagram-get-lambda.invoke_arn
}

# diagram-get-list
output "diagram_get_list_function_name" {
    value = module.diagram-get-list-lambda.function_name
}
output "diagram_get_list_invoke_arn" {
    value = module.diagram-get-list-lambda.invoke_arn
}

# diagram-update
output "diagram_update_function_name" {
    value = module.diagram-update-lambda.function_name
}
output "diagram_update_invoke_arn" {
    value = module.diagram-update-lambda.invoke_arn
}

# diagram-delete
output "diagram_delete_function_name" {
    value = module.diagram-delete-lambda.function_name
}
output "diagram_delete_invoke_arn" {
    value = module.diagram-delete-lambda.invoke_arn
}
