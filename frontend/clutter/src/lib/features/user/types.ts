//to do this need to properly reflect the changes from google auth

export interface UserTenant {
  tenantId: string;
  name: string;
}

export interface UserData {
  userId: string;
  tenantId: string | null;
  email: string;
  displayName: string;
  tenant?: UserTenant;
}
