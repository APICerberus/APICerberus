import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApiRequest } from "@/lib/api";
import type { CreditOverview, CreditTransactionList } from "@/lib/types";
import { normalizeCreditTransactionList, toQueryRecord } from "./helpers";
import { queryKeys } from "./query-keys";

export type CreditTransactionParams = {
  type?: string;
  limit?: number;
  offset?: number;
};

export function useCreditsOverview() {
  return useQuery({
    queryKey: queryKeys.creditsOverview,
    queryFn: () => adminApiRequest<CreditOverview>("/admin/api/v1/credits/overview"),
  });
}

export function useUserCreditBalance(userID: string) {
  return useQuery({
    queryKey: queryKeys.creditsBalance(userID),
    queryFn: () => adminApiRequest<{ balance: number }>(`/admin/api/v1/users/${userID}/credits/balance`),
    enabled: Boolean(userID),
  });
}

export function useUserCreditTransactions(userID: string, params: CreditTransactionParams = {}) {
  const normalizedParams = toQueryRecord(params);

  return useQuery({
    queryKey: queryKeys.creditsTransactions(userID, normalizedParams),
    queryFn: async () => {
      const payload = await adminApiRequest<unknown>(`/admin/api/v1/users/${userID}/credits/transactions`, {
        query: normalizedParams,
      });
      return normalizeCreditTransactionList(payload) as CreditTransactionList;
    },
    enabled: Boolean(userID),
  });
}

export function useTopupCredits() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ userID, amount, reason }: { userID: string; amount: number; reason?: string }) =>
      adminApiRequest<Record<string, unknown>>(`/admin/api/v1/users/${userID}/credits/topup`, {
        method: "POST",
        body: { amount, reason },
      }),
    onSuccess: async (_data, variables) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.creditsOverview });
      await queryClient.invalidateQueries({ queryKey: queryKeys.creditsBalance(variables.userID) });
      await queryClient.invalidateQueries({ queryKey: queryKeys.creditsTransactions(variables.userID) });
      await queryClient.invalidateQueries({ queryKey: queryKeys.users() });
      await queryClient.invalidateQueries({ queryKey: queryKeys.user(variables.userID) });
    },
  });
}

export function useDeductCredits() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ userID, amount, reason }: { userID: string; amount: number; reason?: string }) =>
      adminApiRequest<Record<string, unknown>>(`/admin/api/v1/users/${userID}/credits/deduct`, {
        method: "POST",
        body: { amount, reason },
      }),
    onSuccess: async (_data, variables) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.creditsOverview });
      await queryClient.invalidateQueries({ queryKey: queryKeys.creditsBalance(variables.userID) });
      await queryClient.invalidateQueries({ queryKey: queryKeys.creditsTransactions(variables.userID) });
      await queryClient.invalidateQueries({ queryKey: queryKeys.users() });
      await queryClient.invalidateQueries({ queryKey: queryKeys.user(variables.userID) });
    },
  });
}

