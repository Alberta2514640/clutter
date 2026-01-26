//to do this need to properly reflect the changes from google auth

export interface UserTenant {
  tenantId: string;
  name: string;
}

export interface UserData {
  userId: string;
  token: string;
  picture_url: string;
  email: string;
  displayName: string;
}
