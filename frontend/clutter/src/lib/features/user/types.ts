export interface UserData {
  userId: string;
  token: string;
  picture_url: string;
  email: string;
  displayName: string;
}

export interface GoogleDataShape {
  token?: string;
  user_data?: {
    uuid?: string;
    email?: string;
    full_name?: string;
    picture_url?: string;
  };
}
