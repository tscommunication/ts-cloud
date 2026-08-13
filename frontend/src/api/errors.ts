import axios from 'axios'

interface APIErrorResponse {
  error?: string
  message?: string
}

export function getAPIErrorMessage(
  error: unknown,
  fallback: string,
): string {
  if (axios.isAxiosError<APIErrorResponse>(error)) {
    return (
      error.response?.data?.error ||
      error.response?.data?.message ||
      fallback
    )
  }

  return fallback
}
