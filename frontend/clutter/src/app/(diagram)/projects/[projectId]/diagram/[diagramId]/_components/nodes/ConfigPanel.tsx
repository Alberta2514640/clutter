"use client";

import { Play, Settings, Trash2, Upload, X } from "lucide-react";
import Image from "next/image";
import { useCallback, useMemo, useState } from "react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { DiagramNode } from "@/lib/features/diagram/types";
import { useUpdateDiagramData } from "@/lib/features/diagram/hooks";
import {
  useDiagramEditor,
  useDiagramEditorActions,
} from "@/lib/features/diagram/uiStore";
import { useSupportedResources } from "@/lib/features/resources/hooks";
import {
  useCreateLambdaCodeUploadUrl,
  useUploadLambdaCodeToS3,
} from "@/lib/features/codeUpload/hooks";
import {
  useCreatePlaybookUploadUrl,
  useSubmitAnsibleJob,
  useUploadPlaybookFileToS3,
} from "@/lib/features/runs/hooks";
import { useMe } from "@/lib/features/user/hooks";

const RESOURCE_NAME_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const HIDDEN_VARIABLE_NAMES = new Set([
  "label",
  "position",
  "x",
  "y",
  "position_x",
  "position_y",
  "x_position",
  "y_position",
  "pos_x",
  "pos_y",
]);

const formatVariableLabel = (name: string) => {
  return name
    .split("_")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
};

const getVariableSelectOptions = (
  resourceLabel: string | undefined,
  variableName: string,
) => {
  if (resourceLabel === "DynamoDB") {
    if (variableName === "billing_mode") {
      return ["PAY_PER_REQUEST", "PROVISIONED"];
    }

    if (variableName === "hash_key_type") {
      return ["S", "N", "B"];
    }
  }

  if (resourceLabel === "Lambda" && variableName === "architecture") {
    return ["arm64", "x86_64"];
  }

  if (resourceLabel === "Lambda" && variableName === "memory_size") {
    return ["128", "256", "512", "1024", "1769"];
  }

  if (resourceLabel === "Lambda" && variableName === "runtime") {
    return [
      "nodejs24.x",
      "nodejs22.x",
      "python3.14",
      "python3.13",
      "python3.12",
      "provided.al2023",
    ];
  }

  return null;
};

const getVariableChecklistOptions = (
  resourceLabel: string | undefined,
  variableName: string,
) => {
  if (resourceLabel === "API Gateway" && variableName === "http_methods") {
    return [
      { label: "GET", value: "GET" },
      { label: "POST", value: "POST" },
      { label: "PUT", value: "PUT" },
      { label: "PATCH", value: "PATCH" },
      { label: "DELETE", value: "DELETE" },
    ];
  }

  return null;
};

const getVariablePlaceholder = (
  resourceLabel: string | undefined,
  variableName: string,
  fallback: string,
) => {
  if (resourceLabel === "S3" && variableName === "resource_name") {
    return "e.g. test-bucket";
  }

  return fallback;
};

const getVariableError = (
  name: string,
  value: unknown,
  required: boolean,
) => {
  const stringValue = typeof value === "string" ? value.trim() : "";

  if (required) {
    const isMissing =
      value === undefined ||
      value === null ||
      (typeof value === "string" && stringValue === "");

    if (isMissing) {
      return "This field is required.";
    }
  }

  if (
    name === "resource_name" &&
    stringValue &&
    !RESOURCE_NAME_PATTERN.test(stringValue)
  ) {
    return "Use lowercase letters, numbers, and hyphens only, for example test-lambda.";
  }

  return null;
};

type ConfigPanelProps = {
  diagramId: string;
  projectId: string;
  orgId: string | null;
  accountAccessRoleId: string | null;
};

export default function ConfigPanel({ diagramId, projectId, orgId, accountAccessRoleId, }: ConfigPanelProps) {
  const editor = useDiagramEditor(diagramId);
  const { markClean, setNodes, setEdges } = useDiagramEditorActions();
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const saveDiagram = useUpdateDiagramData(token);
  const createPlaybookUploadUrl = useCreatePlaybookUploadUrl(token);
  const uploadPlaybookFileToS3 = useUploadPlaybookFileToS3();
  const submitAnsibleJob = useSubmitAnsibleJob(token);
  const createLambdaCodeUploadUrl = useCreateLambdaCodeUploadUrl(token);
  const uploadLambdaCodeToS3 = useUploadLambdaCodeToS3();
  const { data: supportedResources } = useSupportedResources();

  const nodes = useMemo(() => editor?.nodes ?? [], [editor?.nodes]);
  const edges = useMemo(() => editor?.edges ?? [], [editor?.edges]);

  const selectedNode = useMemo(() => nodes.find((n) => n.selected), [nodes]);
  const showContent = !!selectedNode;
  const isEc2Node = selectedNode?.data.img?.includes("ec2") ?? false;
  const isLambdaNode = selectedNode?.data.img?.includes("lambda") ?? false;
  const ansibleUploadInputId = selectedNode
    ? `ansible-playbook-upload-${selectedNode.id}`
    : "ansible-playbook-upload";
  const lambdaUploadInputId = selectedNode
    ? `lambda-code-upload-${selectedNode.id}`
    : "lambda-code-upload";

  const [uploadError, setUploadError] = useState<string | null>(null);
  const [uploadMessage, setUploadMessage] = useState<string | null>(null);
  const [runError, setRunError] = useState<string | null>(null);
  const [runMessage, setRunMessage] = useState<string | null>(null);
  const [lambdaUploadError, setLambdaUploadError] = useState<string | null>(null);
  const [lambdaUploadMessage, setLambdaUploadMessage] = useState<string | null>(null);

  const resourceDef = useMemo(
    () => supportedResources?.find((r) => r.label === selectedNode?.data.label),
    [supportedResources, selectedNode?.data.label],
  );

  const visibleVariables = useMemo(
    () =>
      resourceDef?.variables.filter(
        (variable) => !HIDDEN_VARIABLE_NAMES.has(variable.name.toLowerCase()),
      ) ?? [],
    [resourceDef?.variables],
  );

  const patchSelectedNode = useCallback(
    (patch: (node: DiagramNode) => DiagramNode) => {
      if (!selectedNode) return;
      const next = nodes.map((n) =>
        n.id === selectedNode.id ? patch(n) : n,
      );
      setNodes(diagramId, next);
    },
    [diagramId, nodes, selectedNode, setNodes],
  );

  const handleClose = () => {
    patchSelectedNode((n) => ({ ...n, selected: false }));
  };

  const handleVariableChange = (varName: string, value: unknown) => {
    patchSelectedNode((n) => ({
      ...n,
      data: {
        ...n.data,
        variables: { ...(n.data.variables ?? {}), [varName]: value },
      },
    }));
  };

  const persistDiagram = useCallback(
    async (nextNodes: DiagramNode[]) => {
      if (!token) return;

      await saveDiagram.mutateAsync({
        projectId,
        diagramId,
        name: editor?.name?.trim() || "Untitled diagram",
        nodes: nextNodes,
        edges,
      });

      markClean(diagramId);
    },
    [diagramId, editor?.name, edges, markClean, projectId, saveDiagram, token],
  );

  const handleAnsiblePlaybookUpload = useCallback(
    async (file: File | null) => {
      if (!file || !selectedNode) return;
      if (!token) {
        setUploadError("You must be signed in before uploading a playbook.");
        return;
      }

      setUploadError(null);
      setUploadMessage(null);
      setRunError(null);
      setRunMessage(null);

      try {
        const uploadTarget = await createPlaybookUploadUrl.mutateAsync({
          file_name: file.name,
          project_id: projectId,
          diagram_id: diagramId,
        });

        const targetUrl =
          (typeof uploadTarget.upload_url === "string" &&
            uploadTarget.upload_url) ||
          (typeof uploadTarget.url === "string" && uploadTarget.url) ||
          "";

        if (!targetUrl) {
          throw new Error(
            "Upload URL response did not include a usable target URL.",
          );
        }

        await uploadPlaybookFileToS3.mutateAsync({
          upload_url: targetUrl,
          file,
        });

        const nextNodes = nodes.map((node) =>
          node.id !== selectedNode.id
            ? node
            : {
                ...node,
                data: {
                  ...node.data,
                  ansiblePlaybookName: file.name,
                  ansiblePlaybookKey:
                    (typeof uploadTarget.playbook_s3_key === "string" &&
                      uploadTarget.playbook_s3_key) ||
                    (typeof uploadTarget.key === "string" && uploadTarget.key) ||
                    node.data.ansiblePlaybookKey,
                  ansiblePlaybookId:
                    (typeof uploadTarget.playbook_id === "string" &&
                      uploadTarget.playbook_id) ||
                    node.data.ansiblePlaybookId,
                },
              },
        );

        setNodes(diagramId, nextNodes);
        await persistDiagram(nextNodes);

        setUploadMessage(`Uploaded "${file.name}" for this EC2 container.`);
      } catch (err) {
        const message =
          err instanceof Error
            ? err.message
            : "Failed to upload Ansible playbook.";
        setUploadError(message);
      }
    },
    [
      createPlaybookUploadUrl,
      diagramId,
      nodes,
      persistDiagram,
      patchSelectedNode,
      projectId,
      selectedNode,
      setNodes,
      token,
      uploadPlaybookFileToS3,
    ],
  );

  const handleLambdaCodeUpload = useCallback(
    async (file: File | null) => {
      if (!file || !selectedNode) return;
      if (!token) {
        setLambdaUploadError("You must be signed in before uploading code.");
        return;
      }

      setLambdaUploadError(null);
      setLambdaUploadMessage(null);

      const resourceName = String(selectedNode.data.variables?.resource_name ?? "");
      const runtime = String(selectedNode.data.variables?.runtime ?? "");

      if (!resourceName) {
        setLambdaUploadError("Set a resource_name on this Lambda node before uploading code.");
        return;
      }

      try {
        const result = await createLambdaCodeUploadUrl.mutateAsync({
          org_id: orgId ?? "",
          project_id: projectId,
          diagram_id: diagramId,
          lambda_resource_name: resourceName,
          runtime,
        });

        await uploadLambdaCodeToS3.mutateAsync({
          upload_url: result.upload_url,
          file,
        });

        patchSelectedNode((n) => ({
          ...n,
          data: {
            ...n.data,
            lambdaCodeZipName: file.name,
            lambdaCodeS3Key: result.s3_key,
            lambdaCodeS3Bucket: result.s3_bucket,
            variables: {
              ...(n.data.variables ?? {}),
              s3_key: result.s3_key,
              s3_bucket: result.s3_bucket,
            },
          },
        }));

        setLambdaUploadMessage(`Uploaded "${file.name}" for this Lambda function.`);
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to upload Lambda code.";
        setLambdaUploadError(message);
      }
    },
    [
      createLambdaCodeUploadUrl,
      diagramId,
      orgId,
      patchSelectedNode,
      projectId,
      selectedNode,
      token,
      uploadLambdaCodeToS3,
    ],
  );

  const handleTargetInstanceIdChange = useCallback(
    (targetInstanceId: string) => {
      patchSelectedNode((n) => ({
        ...n,
        data: {
          ...n.data,
          ansibleTargetInstanceId: targetInstanceId,
        },
      }));
    },
    [patchSelectedNode],
  );

  const handleRunPlaybook = useCallback(async () => {
    if (!selectedNode) return;
    if (!token) {
      setRunError("You must be signed in before submitting an Ansible job.");
      return;
    }

    const playbookId = selectedNode.data.ansiblePlaybookId?.trim();
    const targetInstanceId = selectedNode.data.ansibleTargetInstanceId?.trim();

    if (!playbookId) {
      setRunError(
        "Upload a playbook first so this EC2 container has a playbook ID.",
      );
      return;
    }

    if (!targetInstanceId) {
      setRunError(
        "Enter the target EC2 instance ID before running the playbook.",
      );
      return;
    }

    if (!accountAccessRoleId) {
      setRunError("Connect an AWS account role before submitting an Ansible job.");
      return;
    }

    setRunError(null);
    setRunMessage(null);

    try {
      const response = await submitAnsibleJob.mutateAsync({
        account_access_role_id: accountAccessRoleId,
        playbook_id: playbookId,
        target_instance_ids: [targetInstanceId],
      });

      patchSelectedNode((n) => ({
        ...n,
        data: {
          ...n.data,
          lastAnsibleJobId: response.job_id,
          lastAnsibleJobStatus: response.status,
        },
      }));

      setRunMessage(
        `Submitted Ansible job ${response.job_id} for this EC2 container.`,
      );
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : "Failed to submit the Ansible job.";
      setRunError(message);
    }
  }, [
    accountAccessRoleId,
    patchSelectedNode,
    selectedNode,
    submitAnsibleJob,
    token,
  ]);

  const handleDeleteSelected = useCallback(() => {
    if (!selectedNode) return;
    const nodeId = selectedNode.id;
    setNodes(diagramId, nodes.filter((n) => n.id !== nodeId));
    setEdges(
      diagramId,
      edges.filter((e) => e.source !== nodeId && e.target !== nodeId),
    );
  }, [diagramId, edges, nodes, selectedNode, setEdges, setNodes]);

  return (
    <aside
      className={[
        "relative h-full shrink-0 border-l border-slate-800 bg-slate-950/70 backdrop-blur",
        "transition-[width] duration-200",
        showContent ? "w-[380px]" : "w-[5px]",
      ].join(" ")}
    >
      {!showContent ? null : (
        <>
          <div className="flex h-14 items-center justify-between gap-2 border-b border-slate-800 px-3">
            <div className="flex min-w-0 items-center gap-2">
              <div className="grid h-8 w-8 place-items-center rounded-lg border border-slate-800 bg-slate-900">
                <Settings className="h-4 w-4 text-slate-300" />
              </div>
              <div className="min-w-0">
                <div className="truncate text-sm font-semibold text-slate-100">
                  {resourceDef?.displayName ?? selectedNode!.data.label}
                </div>
                <div className="truncate text-[11px] text-slate-400">
                  Configuration
                </div>
              </div>
            </div>
            <button
              onClick={handleClose}
              className="grid h-8 w-8 place-items-center rounded-md border border-slate-800 bg-slate-900 transition hover:bg-slate-800"
              title="Deselect node"
              type="button"
            >
              <X className="h-4 w-4 text-slate-300" />
            </button>
          </div>

          <div className="flex h-[calc(100%-3.5rem)] flex-col">
            <div className="flex-1 space-y-4 overflow-y-auto p-4">
              <div className="flex items-center gap-3 rounded-lg border border-slate-800 bg-slate-900/40 p-3">
                <div className="grid h-10 w-10 shrink-0 place-items-center rounded-md border border-slate-800 bg-slate-950">
                  <Image
                    src={selectedNode!.data.img as string}
                    alt=""
                    width={24}
                    height={24}
                    unoptimized
                  />
                </div>
                <div className="min-w-0">
                  <div className="text-xs text-gray-400">Resource</div>
                  <div className="truncate text-sm font-medium text-white">
                    {resourceDef?.displayName ?? selectedNode!.data.label}
                  </div>
                </div>
              </div>

              {resourceDef && visibleVariables.length > 0 && (
                <div>
                  <label className="mb-2 block text-xs font-semibold uppercase tracking-wider text-gray-500">
                    Configuration
                  </label>
                  <div className="space-y-3">
                    {visibleVariables.map((v) => {
                      const val = selectedNode!.data.variables?.[v.name];
                      const placeholder = getVariablePlaceholder(
                        resourceDef?.label,
                        v.name,
                        v.default != null ? String(v.default) : "",
                      );
                      const fieldError = getVariableError(
                        v.name,
                        val,
                        v.required,
                      );
                      const labelText = formatVariableLabel(v.name);
                      const selectOptions = getVariableSelectOptions(
                        resourceDef?.label,
                        v.name,
                      );
                      const checklistOptions = getVariableChecklistOptions(
                        resourceDef?.label,
                        v.name,
                      );
                      const selectedChecklistValues = new Set(
                        String(
                          val ??
                            (typeof v.default === "string" ? v.default : ""),
                        )
                          .split(",")
                          .map((item) => item.trim())
                          .filter(Boolean),
                      );

                      return (
                        <div
                          key={v.name}
                          className="rounded-xl border border-slate-800 bg-slate-900/60 p-3"
                        >
                          <div className="mb-2 flex items-center gap-1 text-xs font-semibold uppercase tracking-wider text-slate-300">
                            <Label>{labelText}</Label>
                            {v.required ? <span className="text-red-400">*</span> : null}
                          </div>

                          {checklistOptions ? (
                            <div className="grid grid-cols-2 gap-2">
                              {checklistOptions.map((option) => {
                                const checked = selectedChecklistValues.has(option.value);

                                return (
                                  <button
                                    key={option.value}
                                    type="button"
                                    onClick={() => {
                                      const nextValues = new Set(selectedChecklistValues);
                                      if (checked) nextValues.delete(option.value);
                                      else nextValues.add(option.value);

                                      handleVariableChange(
                                        v.name,
                                        Array.from(nextValues).join(","),
                                      );
                                    }}
                                    className={[
                                      "flex h-10 items-center gap-2 rounded-lg border px-3 text-sm transition",
                                      checked
                                        ? "border-teal-500/50 bg-teal-500/10 text-teal-100"
                                        : "border-slate-700 bg-slate-950/70 text-slate-300 hover:bg-slate-900/80",
                                    ].join(" ")}
                                  >
                                    <span
                                      className={[
                                        "grid h-4 w-4 place-items-center rounded border text-[10px] font-bold",
                                        checked
                                          ? "border-teal-400 bg-teal-500 text-slate-950"
                                          : "border-slate-600 bg-slate-900 text-transparent",
                                      ].join(" ")}
                                    >
                                      ✓
                                    </span>
                                    <span className="font-medium">{option.label}</span>
                                  </button>
                                );
                              })}
                            </div>
                          ) : v.type === "boolean" ? (
                            <button
                              type="button"
                              role="switch"
                              aria-checked={Boolean(val)}
                              onClick={() =>
                                handleVariableChange(v.name, !Boolean(val))
                              }
                              className={[
                                (
                                  (resourceDef?.label === "API Gateway" &&
                                    v.name === "enable_cors") ||
                                  (resourceDef?.label === "S3" &&
                                    (v.name === "enable_versioning" ||
                                      v.name === "block_public_access")) ||
                                  (resourceDef?.label === "DynamoDB" &&
                                    v.name === "enable_ttl")
                                )
                                  ? "flex h-10 w-full items-center justify-between rounded-lg border px-3 text-sm transition"
                                  : "inline-flex h-10 min-w-[148px] items-center justify-between rounded-lg border px-3 text-sm transition",
                                fieldError
                                  ? "border-red-500/50 bg-slate-950/70 text-red-100"
                                  : "border-slate-700 bg-slate-950/70 text-white",
                              ].join(" ")}
                            >
                              <span className="font-medium">
                                {Boolean(val) ? "True" : "False"}
                              </span>
                              <span
                                className={[
                                  "relative h-5 w-10 rounded-full transition-colors",
                                  Boolean(val)
                                    ? "bg-teal-500/80"
                                    : "bg-slate-700",
                                ].join(" ")}
                              >
                                <span
                                  className={[
                                    "absolute left-0.5 top-0.5 h-4 w-4 rounded-full bg-white transition-transform",
                                    Boolean(val) ? "translate-x-[20px]" : "translate-x-0",
                                  ].join(" ")}
                                />
                              </span>
                            </button>
                          ) : selectOptions ? (
                            <Select
                              value={val === undefined ? "" : String(val)}
                              onValueChange={(nextValue) =>
                                handleVariableChange(v.name, nextValue)
                              }
                            >
                              <SelectTrigger
                                className={[
                                  "h-10 w-full rounded-lg bg-slate-950/70 text-sm text-white",
                                  fieldError
                                    ? "border-red-500/50 focus:ring-red-500/40"
                                    : "border-slate-700 focus:ring-teal-500/40",
                                ].join(" ")}
                              >
                                <SelectValue
                                  placeholder={placeholder || "Select an option"}
                                />
                              </SelectTrigger>
                              <SelectContent>
                                {selectOptions.map((option) => (
                                  <SelectItem key={option} value={option}>
                                    {option}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                          ) : (
                            <Input
                              type={v.type === "number" ? "number" : "text"}
                              className={[
                                "h-10 rounded-lg bg-slate-950/70 text-sm text-white placeholder:text-slate-500",
                                fieldError
                                  ? "border-red-500/50 focus-visible:ring-red-500/40"
                                  : "border-slate-700 focus-visible:ring-teal-500/40",
                              ].join(" ")}
                              placeholder={placeholder || v.description}
                              value={val === undefined ? "" : String(val)}
                              onChange={(e) => {
                                const raw = e.target.value;
                                handleVariableChange(
                                  v.name,
                                  raw === ""
                                    ? undefined
                                    : v.type === "number"
                                      ? Number(raw)
                                      : raw,
                                );
                              }}
                            />
                          )}

                          {v.description &&
                            !(
                              (resourceDef?.label === "DynamoDB" &&
                                (
                                  v.name === "billing_mode" ||
                                  v.name === "hash_key_type" ||
                                  v.name === "enable_ttl"
                                )) ||
                              (resourceDef?.label === "API Gateway" &&
                                v.name === "http_methods")
                            ) && (
                            <p className="mt-2 text-[11px] text-slate-500">
                              {v.description}
                            </p>
                          )}

                          {fieldError && (
                            <p className="mt-2 text-[11px] text-red-300">
                              {fieldError}
                            </p>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}

              {isEc2Node && (
                <div>
                  <div className="mb-4 border-t border-slate-800/80" />
                  <div className="mb-2 block text-xs font-semibold uppercase tracking-wider text-gray-500">
                    Ansible Playbook
                  </div>
                  <div className="space-y-3 rounded-lg border border-cyan-900/40 bg-cyan-950/20 p-3">
                    <div className="rounded-lg border border-cyan-900/40 bg-slate-950/75 px-3 py-2">
                      <div className="text-xs text-gray-400">
                        Bound EC2 container
                      </div>
                      <div className="mt-1 text-sm font-medium text-white">
                        {selectedNode!.data.label}
                      </div>
                    </div>

                    <label
                      htmlFor={ansibleUploadInputId}
                      className="flex cursor-pointer items-center justify-center gap-2 rounded-lg border border-dashed border-cyan-800/60 bg-slate-950/65 px-3 py-3 text-sm font-medium text-cyan-50 transition hover:border-cyan-500/60 hover:bg-cyan-950/30"
                    >
                      <Upload className="h-4 w-4" />
                      {selectedNode!.data.ansiblePlaybookName
                        ? "Replace playbook for this EC2 container"
                        : "Upload playbook for this EC2 container"}
                    </label>

                    <input
                      id={ansibleUploadInputId}
                      type="file"
                      accept=".yml,.yaml"
                      className="hidden"
                      onChange={async (e) => {
                        const input = e.currentTarget;
                        const file = input.files?.[0] ?? null;
                        await handleAnsiblePlaybookUpload(file);
                        input.value = "";
                      }}
                    />

                    <div className="rounded-lg border border-cyan-900/40 bg-slate-950/75 px-3 py-2">
                      <div className="text-xs text-gray-400">Uploaded file</div>
                      <div className="mt-1 text-sm font-medium text-white">
                        {selectedNode!.data.ansiblePlaybookName ??
                          "No playbook uploaded yet"}
                      </div>
                    </div>

                    <div>
                      <label className="mb-2 block text-xs font-semibold uppercase tracking-wider text-gray-500">
                        Target EC2 instance ID
                      </label>
                      <input
                        type="text"
                        value={selectedNode!.data.ansibleTargetInstanceId ?? ""}
                        onChange={(e) =>
                          handleTargetInstanceIdChange(e.target.value)
                        }
                        placeholder="i-0a05920cb52c4555d"
                        className="w-full rounded-lg border border-cyan-900/40 bg-slate-950/75 px-3 py-2 text-sm text-white transition-colors focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
                      />
                      <div className="mt-1 text-xs text-slate-400">
                        This instance ID is stored on this EC2 container node
                        and used as the Ansible target.
                      </div>
                    </div>

                    <div className="rounded-lg border border-cyan-900/40 bg-slate-950/75 px-3 py-2">
                      <div className="text-xs text-gray-400">Playbook ID</div>
                      <div className="mt-1 break-all text-sm font-medium text-white">
                        {selectedNode!.data.ansiblePlaybookId ??
                          "Not available until upload succeeds"}
                      </div>
                    </div>

                    <button
                      type="button"
                      onClick={handleRunPlaybook}
                      disabled={
                        !accountAccessRoleId ||
                        !selectedNode!.data.ansiblePlaybookId ||
                        !selectedNode!.data.ansibleTargetInstanceId?.trim() ||
                        createPlaybookUploadUrl.isPending ||
                        uploadPlaybookFileToS3.isPending ||
                        submitAnsibleJob.isPending
                      }
                      className={[
                        "flex w-full items-center justify-center gap-2 rounded-lg border px-3 py-2 text-sm font-medium transition",
                        selectedNode!.data.ansiblePlaybookId &&
                        selectedNode!.data.ansibleTargetInstanceId?.trim()
                          ? "border-teal-500/40 bg-teal-500/10 text-teal-100 hover:bg-teal-500/20"
                          : "border-slate-800 bg-slate-950/80 text-slate-500",
                      ].join(" ")}
                    >
                      <Play className="h-4 w-4" />
                      {submitAnsibleJob.isPending
                        ? "Submitting Ansible job..."
                        : !accountAccessRoleId
                          ? "Connect AWS account to run"
                          : selectedNode!.data.ansiblePlaybookId
                            ? "Run playbook on this EC2 container"
                            : "Upload a playbook to run"}
                    </button>

                    {(createPlaybookUploadUrl.isPending ||
                      uploadPlaybookFileToS3.isPending) && (
                      <div className="text-xs text-slate-400">
                        Uploading playbook…
                      </div>
                    )}

                    {submitAnsibleJob.isPending && (
                      <div className="text-xs text-slate-400">
                        Submitting Ansible job…
                      </div>
                    )}

                    {uploadMessage && (
                      <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/10 px-3 py-2 text-xs text-emerald-200">
                        {uploadMessage}
                      </div>
                    )}

                    {uploadError && (
                      <div className="rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-2 text-xs text-red-200">
                        {uploadError}
                      </div>
                    )}

                    {runMessage && (
                      <div className="rounded-lg border border-cyan-500/20 bg-cyan-500/10 px-3 py-2 text-xs text-cyan-200">
                        {runMessage}
                      </div>
                    )}

                    {runError && (
                      <div className="rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-2 text-xs text-red-200">
                        {runError}
                      </div>
                    )}
                  </div>
                </div>
              )}

              {isLambdaNode && (
                <div>
                  <div className="mb-4 border-t border-slate-800/80" />
                  <div className="mb-2 block text-xs font-semibold uppercase tracking-wider text-gray-500">
                    Function Code
                  </div>
                  <div className="space-y-3 rounded-lg border border-cyan-900/40 bg-cyan-950/20 p-3">
                    <label
                      htmlFor={lambdaUploadInputId}
                      className="flex cursor-pointer items-center justify-center gap-2 rounded-lg border border-dashed border-cyan-800/60 bg-slate-950/65 px-3 py-3 text-sm font-medium text-cyan-50 transition hover:border-cyan-500/60 hover:bg-cyan-950/30"
                    >
                      <Upload className="h-4 w-4" />
                      {selectedNode!.data.lambdaCodeZipName
                        ? "Replace function code (.zip)"
                        : "Upload function code (.zip)"}
                    </label>

                    <input
                      id={lambdaUploadInputId}
                      type="file"
                      accept=".zip"
                      className="hidden"
                      onChange={async (e) => {
                        const input = e.currentTarget;
                        await handleLambdaCodeUpload(
                          e.target.files?.[0] ?? null,
                        );
                        input.value = "";
                      }}
                    />

                    <div className="rounded-lg border border-cyan-900/40 bg-slate-950/75 px-3 py-2">
                      <div className="text-xs text-gray-400">Uploaded file</div>
                      <div className="mt-1 text-sm font-medium text-white">
                        {selectedNode!.data.lambdaCodeZipName ?? "No file uploaded yet"}
                      </div>
                    </div>

                    {(createLambdaCodeUploadUrl.isPending ||
                      uploadLambdaCodeToS3.isPending) && (
                      <div className="text-xs text-slate-400">
                        Uploading function code…
                      </div>
                    )}

                    {lambdaUploadMessage && (
                      <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/10 px-3 py-2 text-xs text-emerald-200">
                        {lambdaUploadMessage}
                      </div>
                    )}

                    {lambdaUploadError && (
                      <div className="rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-2 text-xs text-red-200">
                        {lambdaUploadError}
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>

            <div className="border-t border-slate-800 p-4">
              <div className="mb-2 text-[11px] font-semibold uppercase tracking-wider text-gray-500">
                Danger zone
              </div>
              <button
                type="button"
                onClick={handleDeleteSelected}
                className={[
                  "flex w-full items-center justify-center gap-2 rounded-lg border px-3 py-2 text-sm font-medium transition",
                  "border-red-900/40 bg-red-950/40 text-red-200",
                  "hover:bg-red-950/60 hover:border-red-800/60",
                  "focus:outline-none focus:ring-1 focus:ring-red-500/60",
                ].join(" ")}
                title="Delete selected node"
              >
                <Trash2 className="h-4 w-4" />
                Delete node
              </button>
              <p className="mt-2 text-xs text-slate-400">
                This will also remove any connections to this node.
              </p>
            </div>
          </div>
        </>
      )}
    </aside>
  );
}
