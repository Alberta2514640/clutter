export type CreateLambdaCodeUploadUrlInput = {
  org_id: string;
  project_id: string;
  diagram_id: string;
  lambda_resource_name: string;
  runtime: string;
};

export type CreateLambdaCodeUploadUrlResponse = {
  upload_url: string;
  s3_bucket: string;
  s3_key: string;
  [key: string]: unknown;
};

export type UploadLambdaCodeToS3Input = {
  upload_url: string;
  file: File;
};
