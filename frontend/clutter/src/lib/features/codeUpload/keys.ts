export const codeUploadKeys = {
  base: ["codeUpload"] as const,
  lambdaUploadUrl: () => [...codeUploadKeys.base, "lambda-upload-url"] as const,
  lambdaS3Upload: () => [...codeUploadKeys.base, "lambda-s3-upload"] as const,
};
