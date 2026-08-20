import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { toast } from "@components/ui/toast";
import type { ApiError, SyncStatus } from "@lib/api";
import { getDirectorySync, triggerDirectorySync, unwrap } from "@lib/api";

const directorySyncKey = ["directory", "sync"] as const;

export function useDirectorySync() {
  return useQuery<SyncStatus, ApiError>({
    queryKey: directorySyncKey,
    queryFn: ({ signal }) => unwrap(getDirectorySync({ signal })),
    refetchInterval: (query) => {
      const status = query.state.data;
      if (!status?.enabled) return false;
      return status.activity === "queued" || status.activity === "running" ? 2_000 : 30_000;
    },
  });
}

export function useTriggerDirectorySync() {
  const queryClient = useQueryClient();

  return useMutation<SyncStatus, ApiError>({
    mutationFn: () => unwrap(triggerDirectorySync()),
    onSuccess: (status) => {
      queryClient.setQueryData(directorySyncKey, status);
      toast.add({
        title:
          status.activity === "running" ? "Directory Sync Is Running" : "Directory Sync Queued",
        type: "success",
      });
    },
    onError: (error) => {
      toast.add({
        title: "Directory sync could not be queued",
        description: error.message,
        type: "error",
      });
    },
  });
}
