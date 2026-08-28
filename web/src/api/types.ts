export type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "HEAD" | "OPTIONS";

export interface KeyValue {
  key: string;
  value: string;
  enabled: boolean;
}

export type BodyType = "none" | "json" | "xml" | "form" | "raw";

export interface RequestBody {
  type: BodyType;
  content?: string;
}

export type AuthType = "none" | "bearer" | "basic" | "apikey";

export interface Auth {
  type: AuthType;
  token?: string;
  username?: string;
  password?: string;
  keyName?: string;
  keyValue?: string;
  in?: "header" | "query";
}

export interface Extraction {
  variable: string;
  from: string;
}

export interface ApiRequest {
  id: string;
  name: string;
  method: HttpMethod;
  url: string;
  headers?: KeyValue[];
  queryParams?: KeyValue[];
  body?: RequestBody;
  auth?: Auth;
  assertions?: string[];
  extract?: Extraction[];
}

export interface Collection {
  id: string;
  name: string;
  description?: string;
  requests: ApiRequest[];
}

export interface Variable {
  value: string;
  secret: boolean;
}

export interface Environment {
  id: string;
  name: string;
  variables: Record<string, Variable>;
}

export interface AssertionResult {
  expression: string;
  passed: boolean;
  message: string;
}

export interface StepResult {
  request: string;
  method: string;
  url: string;
  status?: number;
  durationMs: number;
  headers?: Record<string, string[]>;
  body?: string;
  assertions?: AssertionResult[];
  passed: boolean;
  error?: string;
}

export interface Report {
  collectionName: string;
  environment?: string;
  steps: StepResult[];
  started: string;
  durationMs: number;
  passed: number;
  failed: number;
}

export interface RunSummary {
  id: string;
  collectionId: string;
  collectionName: string;
  environmentName: string;
  report: Report;
  passed: number;
  failed: number;
  startedAt: string;
  durationMs: number;
}

export interface HistoryEntry {
  id: string;
  requestName: string;
  method: string;
  url: string;
  status: number;
  durationMs: number;
  executedAt: string;
}

export type WsEvent =
  | { type: "step"; step: StepResult }
  | { type: "done"; report: Report }
  | { type: "error"; error: string };
