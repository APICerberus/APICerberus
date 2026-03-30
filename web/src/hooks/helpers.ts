import type { CreditTransaction, CreditTransactionList, User, UserListResponse } from "@/lib/types";

export function toQueryRecord(params: Record<string, unknown>) {
  const result: Record<string, string | number | boolean> = {};
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") {
      continue;
    }
    if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
      result[key] = value;
    }
  }
  return result;
}

export function normalizeUserListResponse(payload: unknown): UserListResponse {
  const source = (payload as Record<string, unknown>) ?? {};
  const users = (source.users ?? source.Users ?? []) as User[];
  const total = Number(source.total ?? source.Total ?? users.length);
  return { users, total };
}

export function normalizeCreditTransactionList(payload: unknown): CreditTransactionList {
  const source = (payload as Record<string, unknown>) ?? {};
  const transactions = (source.transactions ?? source.Transactions ?? []) as CreditTransaction[];
  const total = Number(source.total ?? source.Total ?? transactions.length);
  return { transactions, total };
}

