// Centralized API client for calling Lambda functions via API Gateway
//old api called to will be changed to stores

interface ApiConfig {
  baseUrl: string;
  apiKey?: string;
  useMockData?: boolean; // Enable mock data mode
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

  private async request<T>(
    endpoint: string,
    options: RequestInit = {},
  ): Promise<T> {
    // Return mock data if enabled
    if (this.useMockData) {
      return this.getMockData<T>(endpoint, options.method || "GET");
    }

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    if (this.apiKey) {
      headers["x-api-key"] = this.apiKey;
    }

    // TODO: Add auth token when Cognito is set up
    // const session = await getSession();
    // if (session?.accessToken) {
    //   headers['Authorization'] = `Bearer ${session.accessToken}`;
    // }

    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new Error(error.message || `API Error: ${response.status}`);
    }

    return response.json();
  }

  // Mock data generator
  private async getMockData<T>(endpoint: string, method: string): Promise<T> {
    // Simulate network delay
    await new Promise((resolve) => setTimeout(resolve, 300));

    // Mock user profile
    if (endpoint === "/user/profile" && method === "GET") {
      return {
        userId: "demo_user_001",
        tenantId: "t_demo_001",
        email: "demo@example.com",
        displayName: "Demo User",
        tenant: {
          tenantId: "t_demo_001",
          name: "Demo Organization",
        },
      } as T;
    }

    // Mock projects list
    if (endpoint === "/projects" && method === "GET") {
      return {
        projects: [
          {
            projectId: "p_web_app",
            name: "Web Application",
            description:
              "Production web application with Lambda, API Gateway, and DynamoDB",
            createdAt: "2025-01-15T10:00:00Z",
            updatedAt: "2025-01-20T14:30:00Z",
            memberCount: 3,
          },
          {
            projectId: "p_data_pipeline",
            name: "Data Pipeline",
            description: "ETL pipeline with S3, Lambda, and Glue",
            createdAt: "2025-01-10T08:00:00Z",
            updatedAt: "2025-01-18T16:45:00Z",
            memberCount: 2,
          },
          {
            projectId: "p_monitoring",
            name: "Monitoring Stack",
            description: "CloudWatch dashboards and alarms",
            createdAt: "2025-01-05T12:00:00Z",
            updatedAt: "2025-01-12T09:15:00Z",
            memberCount: 1,
          },
        ],
      } as T;
    }

    // Mock recent runs
    if (endpoint === "/runs/recent" && method === "GET") {
      return {
        runs: [
          {
            runId: "run_001",
            projectId: "p_web_app",
            projectName: "Web Application",
            workspaceId: "ws_prod",
            action: "apply",
            status: "SUCCESS",
            startedAt: "2025-01-20T14:00:00Z",
            endedAt: "2025-01-20T14:05:30Z",
          },
          {
            runId: "run_002",
            projectId: "p_web_app",
            projectName: "Web Application",
            workspaceId: "ws_staging",
            action: "plan",
            status: "SUCCESS",
            startedAt: "2025-01-19T10:30:00Z",
            endedAt: "2025-01-19T10:32:15Z",
          },
          {
            runId: "run_003",
            projectId: "p_data_pipeline",
            projectName: "Data Pipeline",
            workspaceId: "ws_prod",
            action: "apply",
            status: "RUNNING",
            startedAt: "2025-01-21T08:15:00Z",
          },
          {
            runId: "run_004",
            projectId: "p_monitoring",
            projectName: "Monitoring Stack",
            workspaceId: "ws_prod",
            action: "apply",
            status: "FAILED",
            startedAt: "2025-01-18T16:00:00Z",
            endedAt: "2025-01-18T16:03:45Z",
          },
        ],
      } as T;
    }

    // Mock single project
    if (
      endpoint.startsWith("/projects/") &&
      !endpoint.includes("/workspaces") &&
      method === "GET"
    ) {
      const projectId = endpoint.split("/")[2];
      return {
        projectId,
        name: "Web Application",
        description: "Production web application",
        createdAt: "2025-01-15T10:00:00Z",
        updatedAt: "2025-01-20T14:30:00Z",
        tenantId: "t_demo_001",
        memberCount: 3,
      } as T;
    }

    // Mock workspaces
    if (endpoint.includes("/workspaces") && method === "GET") {
      return {
        workspaces: [
          {
            workspaceId: "ws_prod",
            name: "Production",
            accountRef: {
              tenantId: "t_demo_001",
              accountId: "123456789012",
              alias: "prod",
            },
          },
          {
            workspaceId: "ws_staging",
            name: "Staging",
            accountRef: {
              tenantId: "t_demo_001",
              accountId: "123456789012",
              alias: "staging",
            },
          },
        ],
      } as T;
    }

    // Mock canvas
    if (endpoint.includes("/canvas/") && method === "GET") {
      return {
        canvasId: endpoint.split("/").pop(),
        name: "Main Architecture",
        nodes: [],
        edges: [],
        uiLayout: { viewport: { x: 0, y: 0, zoom: 1 }, nodes: {} },
      } as T;
    }

    // Default empty response
    return {} as T;
  }

  // User endpoints
  async getUserProfile() {
    return this.request<{
      userId: string;
      tenantId: string | null;
      email: string;
      displayName: string;
      tenant?: { tenantId: string; name: string };
    }>("/user/profile", { method: "GET" });
  }

  // Project endpoints
  async getProjects() {
    return this.request<{
      projects: Array<{
        projectId: string;
        name: string;
        description: string;
        createdAt: string;
        updatedAt: string;
        memberCount?: number;
      }>;
    }>("/projects", { method: "GET" });
  }

  async getProject(projectId: string) {
    return this.request(`/projects/${projectId}`, { method: "GET" });
  }

  async createProject(data: { name: string; description: string }) {
    return this.request("/projects", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async updateProject(
    projectId: string,
    data: Partial<{ name: string; description: string }>,
  ) {
    return this.request(`/projects/${projectId}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  }

  async deleteProject(projectId: string) {
    return this.request(`/projects/${projectId}`, { method: "DELETE" });
  }

  // Run endpoints
  async getRecentRuns() {
    return this.request<{
      runs: Array<{
        runId: string;
        projectId: string;
        projectName: string;
        workspaceId: string;
        action: "plan" | "apply";
        status: "QUEUED" | "RUNNING" | "SUCCESS" | "FAILED";
        startedAt: string;
        endedAt?: string;
      }>;
    }>("/runs/recent", { method: "GET" });
  }

  async getRun(projectId: string, runId: string) {
    return this.request(`/projects/${projectId}/runs/${runId}`, {
      method: "GET",
    });
  }

  // Workspace endpoints
  async getWorkspaces(projectId: string) {
    return this.request(`/projects/${projectId}/workspaces`, { method: "GET" });
  }

  async createWorkspace(projectId: string, data: any) {
    return this.request(`/projects/${projectId}/workspaces`, {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  // Canvas endpoints
  async getCanvas(projectId: string, canvasId: string) {
    return this.request(`/projects/${projectId}/canvas/${canvasId}`, {
      method: "GET",
    });
  }

  async updateCanvas(projectId: string, canvasId: string, data: any) {
    return this.request(`/projects/${projectId}/canvas/${canvasId}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  }

  // Tenant endpoints
  async createTenant(data: { name: string; tags?: string[] }) {
    return this.request("/tenant", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async getTenantMembers() {
    return this.request("/tenant/members", { method: "GET" });
  }
}

// Export singleton instance with mock data enabled
export const apiClient = new ApiClient({
  baseUrl: process.env.NEXT_PUBLIC_API_URL || "https://api.your-domain.com",
  apiKey: process.env.NEXT_PUBLIC_API_KEY,
  useMockData:
    process.env.NEXT_PUBLIC_USE_MOCK_DATA === "true" ||
    !process.env.NEXT_PUBLIC_API_URL, // Auto-enable if no API URL
});
