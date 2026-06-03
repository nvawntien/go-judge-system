export type ApiResponse<T> = {
  status: "success" | "error";
  code: number;
  msg?: string;
  data?: T;
};
