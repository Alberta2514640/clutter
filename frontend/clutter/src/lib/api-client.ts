// Centralized API client for calling Lambda functions via API Gateway
import type { Canvas, Node, Edge, Workspace, Run, RunStatus, CanvasBundle } from "@/lib/types";

interface ApiConfig {
  baseUrl: string;
  apiKey?: string;
  useMockData?: boolean; // Enable mock data mode
}

// User type matching database spec
export interface User {
  userId: string;
  organizationId: string;
  email: string;
  displayName: string;
  pictureUrl: string;
  createdAt: string;
}

// API Response from login endpoint
interface LoginResponse {
  token: string;
  userData: {
    uuid: string;
    email: string;
    name: string;
    pictureUrl: string;
    accountCreatedOn: string;
  };
  message?: string;
}

class ApiClient {
  private baseUrl: string;
  private apiKey?: string;
  private useMockData: boolean;

  constructor(config: ApiConfig) {
    this.baseUrl = config.baseUrl;
    this.apiKey = config.apiKey;
    this.useMockData = config.useMockData || false;
  }

  // ============================================================================
  // HELPER METHODS
  // ============================================================================

  /**
   * Get auth token from localStorage
   */
  private getAuthToken(): string | null {
    if (typeof window === "undefined") return null;
    return localStorage.getItem("clutter_auth_token");
  }

  /**
   * Set auth token in localStorage
   */
  private setAuthToken(token: string): void {
    if (typeof window !== "undefined") {
      localStorage.setItem("clutter_auth_token", token);
    }
  }

  /**
   * Clear auth token from localStorage
   */
  private clearAuthToken(): void {
    if (typeof window !== "undefined") {
      localStorage.removeItem("clutter_auth_token");
    }
  }

  /**
   * Map API user response to database spec
   */
  private mapUserData(apiUser: LoginResponse["userData"]): User {
    return {
      userId: apiUser.uuid,
      organizationId: apiUser.uuid, // Using uuid as organizationId
      email: apiUser.email,
      displayName: apiUser.name,
      pictureUrl: apiUser.pictureUrl,
      createdAt: apiUser.accountCreatedOn,
    };
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    // Return mock data if enabled
    if (this.useMockData) {
      return this.getMockData<T>(endpoint, options.method || 'GET');
    }

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (this.apiKey) {
      headers['x-api-key'] = this.apiKey;
    }

    // Add auth token to requests
    const authToken = this.getAuthToken();
    if (authToken) {
      headers['Authorization'] = `${authToken}`;
    }

    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      // If unauthorized, clear token
      if (response.status === 401) {
        this.clearAuthToken();
      }

      const error = await response.json().catch(() => ({}));
      throw new Error(error.message || `API Error: ${response.status}`);
    }

    const json = await response.json();
    
    // Unwrap the data field from API response
    // API returns: { data: [...], success: true }
    // We return just the data field
    if (json && typeof json === 'object' && 'data' in json) {
      return json.data as T;
    }

    // If no data field, return as-is (for backward compatibility)
    return json as T;
  }

  // Mock data generator
  private async getMockData<T>(endpoint: string, method: string): Promise<T> {
    // Simulate network delay
    await new Promise(resolve => setTimeout(resolve, 300));

    // Mock login - needed for development
    if (endpoint === '/log-in' && method === 'POST') {
      return {
        token: 'mock_jwt_token_' + Date.now(),
        userData: {
          uuid: 'mock_user_001',
          email: 'demo@clutter.com',
          name: 'Demo User',
          pictureUrl: 'https://avatar.vercel.sh/demo',
          accountCreatedOn: new Date().toISOString(),
        }
      } as T;
    }

    // Mock current user - needed for auth check (TODO: Build /me endpoint)
    if (endpoint === '/me' && method === 'GET') {
      const token = this.getAuthToken();
      if (!token) {
        throw new Error('Unauthorized');
      }
      return {
        uuid: 'mock_user_001',
        email: 'demo@clutter.com',
        name: 'Demo User',
        pictureUrl: 'https://avatar.vercel.sh/demo',
        accountCreatedOn: '2025-01-01T00:00:00Z',
      } as T;
    }

    // Mock recent runs (TODO: Build /runs/recent endpoint)
    if (endpoint === '/runs/recent' && method === 'GET') {
      return {
        runs: [
          {
            runId: 'run_001',
            projectId: 'p_web_app',
            projectName: 'Web Application',
            workspaceId: 'ws_prod',
            action: 'apply',
            status: 'SUCCEEDED' as RunStatus,
            startedAt: '2025-01-20T14:00:00Z',
            endedAt: '2025-01-20T14:05:30Z'
          },
          {
            runId: 'run_002',
            projectId: 'p_web_app',
            projectName: 'Web Application',
            workspaceId: 'ws_staging',
            action: 'plan',
            status: 'SUCCEEDED' as RunStatus,
            startedAt: '2025-01-19T10:30:00Z',
            endedAt: '2025-01-19T10:32:15Z'
          },
          {
            runId: 'run_003',
            projectId: 'p_data_pipeline',
            projectName: 'Data Pipeline',
            workspaceId: 'ws_prod',
            action: 'apply',
            status: 'PLAN' as RunStatus,
            startedAt: '2025-01-21T08:15:00Z'
          },
          {
            runId: 'run_004',
            projectId: 'p_monitoring',
            projectName: 'Monitoring Stack',
            workspaceId: 'ws_prod',
            action: 'apply',
            status: 'FAILED' as RunStatus,
            startedAt: '2025-01-18T16:00:00Z',
            endedAt: '2025-01-18T16:03:45Z'
          }
        ]
      } as T;
    }

    // NOTE: /projects now uses REAL API (not mocked)

    // Mock recent runs
    if (endpoint === '/runs/recent' && method === 'GET') {
      return {
        runs: [
          {
            runId: 'run_001',
            projectId: 'p_web_app',
            projectName: 'Web Application',
            workspaceId: 'ws_prod',
            action: 'apply',
            status: 'SUCCEEDED' as RunStatus,
            startedAt: '2025-01-20T14:00:00Z',
            endedAt: '2025-01-20T14:05:30Z'
          },
          {
            runId: 'run_002',
            projectId: 'p_web_app',
            projectName: 'Web Application',
            workspaceId: 'ws_staging',
            action: 'plan',
            status: 'SUCCEEDED' as RunStatus,
            startedAt: '2025-01-19T10:30:00Z',
            endedAt: '2025-01-19T10:32:15Z'
          },
          {
            runId: 'run_003',
            projectId: 'p_data_pipeline',
            projectName: 'Data Pipeline',
            workspaceId: 'ws_prod',
            action: 'apply',
            status: 'PLAN' as RunStatus,
            startedAt: '2025-01-21T08:15:00Z'
          },
          {
            runId: 'run_004',
            projectId: 'p_monitoring',
            projectName: 'Monitoring Stack',
            workspaceId: 'ws_prod',
            action: 'apply',
            status: 'FAILED' as RunStatus,
            startedAt: '2025-01-18T16:00:00Z',
            endedAt: '2025-01-18T16:03:45Z'
          }
        ]
      } as T;
    }

    // Mock single project
    if (endpoint.startsWith('/projects/') && !endpoint.includes('/workspaces') && method === 'GET') {
      const projectId = endpoint.split('/')[2];
      return {
        projectId,
        name: 'Web Application',
        description: 'Production web application',
        createdAt: '2025-01-15T10:00:00Z',
        updatedAt: '2025-01-20T14:30:00Z',
        tenantId: 't_demo_001',
        memberCount: 3
      } as T;
    }

    // Mock workspaces
    if (endpoint.includes('/workspaces') && method === 'GET') {
      return {
        workspaces: [
          {
            workspaceId: 'ws_prod',
            name: 'Production',
            accountRef: { tenantId: 't_demo_001', accountId: '123456789012', alias: 'prod' }
          },
          {
            workspaceId: 'ws_staging',
            name: 'Staging',
            accountRef: { tenantId: 't_demo_001', accountId: '123456789012', alias: 'staging' }
          }
        ]
      } as T;
    }

    // Mock canvas
    if (endpoint.includes('/canvas/') && method === 'GET') {
      return {
        canvas: {
          canvasId: endpoint.split('/').pop() || 'canvas_1',
          name: 'Main Architecture',
          uiLayout: { viewport: { x: 0, y: 0, zoom: 1 }, nodes: {} }
        },
        nodes: [],
        edges: []
      } as T;
    }

    // Default empty response
    return {} as T;
  }

  // ============================================================================
  // AUTH ENDPOINTS
  // ============================================================================

  /**
   * Login with Google ID token
   */
  async login(googleIdToken: string): Promise<{ user: User; token: string }> {
    const response = await this.request<LoginResponse>('/log-in', {
      method: 'POST',
      body: JSON.stringify({ token: googleIdToken }),
    });

    // Store token in localStorage
    this.setAuthToken(response.token);

    // Map user data and return
    const user = this.mapUserData(response.userData);

    return {
      user,
      token: response.token,
    };
  }

  /**
   * Get current authenticated user
   * Returns null if not authenticated
   */
  async getCurrentUser(): Promise<User | null> {
    const token = this.getAuthToken();

    if (!token) {
      return null;
    }

    try {
      // TODO: Update this endpoint to match your backend
      // For now using /me - update to your actual endpoint
      const response = await this.request<LoginResponse["userData"]>('/me', {
        method: 'GET',
      });

      return this.mapUserData(response);
    } catch (error) {
      // If request fails, clear token and return null
      this.clearAuthToken();
      return null;
    }
  }

  /**
   * Logout - clear auth token
   */
  async logout(): Promise<void> {
    this.clearAuthToken();

    // Optional: call server-side logout if you have one
    // await this.request('/logout', { method: 'POST' });
  }

  // ============================================================================
  // USER ENDPOINTS
  // ============================================================================

  async getUserProfile() {
    return this.request<{
      userId: string;
      tenantId: string | null;
      email: string;
      displayName: string;
      tenant?: { tenantId: string; name: string };
    }>('/user/profile', { method: 'GET' });
  }

  // ============================================================================
  // PROJECT ENDPOINTS
  // ============================================================================

  async getProjects(organizationId: string) {
    const data = await this.request<Array<{
      id: string;
      organizationId: string;
      name: string;
      description: string;
      createdBy: string;
      createdAt: string;
      updatedAt?: string;
    }>>(`/projects?organizationId=${organizationId}`, { method: 'GET' });


    console.log("Fetched projects:", data);
    // Map API response to expected format
    return {
      projects: data.map(project => ({
        projectId: project.id,
        name: project.name,
        description: project.description,
        createdAt: project.createdAt,
        updatedAt: project.updatedAt || project.createdAt,
        memberCount: undefined, // Not provided by API yet
      }))
    };
  }

  async getProject(organizationId: string, projectId: string) {
    return this.request(`/projects?organizationId=${organizationId}&projectId=${projectId}`, { method: 'GET' });
  }

  async createProject(organizationId: string, data: { name: string; description: string }) {
    return this.request('/projects', {
      method: 'POST',
      body: JSON.stringify({
        ...data,
        organizationId,
      }),
    });
  }

  async updateProject(projectId: string, data: Partial<{ name: string; description: string }>) {
    return this.request(`/projects/${projectId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteProject(projectId: string) {
    return this.request(`/projects/${projectId}`, { method: 'DELETE' });
  }

  // ============================================================================
  // RUN ENDPOINTS
  // ============================================================================

  async getRecentRuns() {
    return this.request<{
      runs: Array<{
        runId: string;
        projectId: string;
        projectName: string;
        workspaceId: string;
        action: 'plan' | 'apply';
        status: RunStatus;
        startedAt: string;
        endedAt?: string;
      }>;
    }>('/runs/recent', { method: 'GET' });
  }

  async getRun(projectId: string, runId: string) {
    return this.request(`/projects/${projectId}/runs/${runId}`, { method: 'GET' });
  }

  // ============================================================================
  // WORKSPACE ENDPOINTS
  // ============================================================================

  async getWorkspaces(projectId: string) {
    return this.request<{
      workspaces: Workspace[];
    }>(`/projects/${projectId}/workspaces`, { method: 'GET' });
  }

  async createWorkspace(projectId: string, data: any) {
    return this.request(`/projects/${projectId}/workspaces`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // ============================================================================
  // CANVAS ENDPOINTS
  // ============================================================================

  async getCanvas(projectId: string, canvasId: string) {
    return this.request<CanvasBundle>(`/projects/${projectId}/canvas/${canvasId}`, { method: 'GET' });
  }

  async updateCanvas(projectId: string, canvasId: string, data: any) {
    return this.request(`/projects/${projectId}/canvas/${canvasId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  // ============================================================================
  // TENANT ENDPOINTS
  // ============================================================================

  async createTenant(data: { name: string; tags?: string[] }) {
    return this.request('/tenant', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getTenantMembers() {
    return this.request('/tenant/members', { method: 'GET' });
  }
}

// Export singleton instance with mock data enabled
export const apiClient = new ApiClient({
  baseUrl: process.env.NEXT_PUBLIC_API_URL || 'https://qzq3ncab46.execute-api.us-west-2.amazonaws.com/prod',
  apiKey: process.env.NEXT_PUBLIC_API_KEY,
  useMockData: process.env.NEXT_PUBLIC_USE_MOCK_DATA === 'true' || !process.env.NEXT_PUBLIC_API_URL,
});