interface LoadingMessageProps {
  message: string
}

export function LoadingMessage({ message }: LoadingMessageProps) {
  return <p>{message}</p>
}
