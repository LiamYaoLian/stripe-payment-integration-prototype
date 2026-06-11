import { z } from 'zod'

export const orderStatusSchema = z.enum([
  'pending',
  'processing',
  'paid',
  'failed',
  'expired',
  'canceled',
  'refunded',
])

export const productSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().optional(),
  unitAmountCents: z.number().int().nonnegative(),
  currency: z.string(),
})

export const orderItemSchema = z.object({
  productName: z.string(),
  quantity: z.number().int().positive(),
  lineTotalCents: z.number().int().nonnegative(),
})

export const orderSchema = z.object({
  id: z.string(),
  orderNumber: z.string(),
  status: orderStatusSchema,
  totalAmountCents: z.number().int().nonnegative(),
  currency: z.string(),
  paidAt: z.string().optional(),
  items: z.array(orderItemSchema),
})

export const userProfileSchema = z.object({
  id: z.string(),
  email: z.string(),
  emailVerified: z.boolean(),
})

export const authSessionSchema = z.object({
  expiresAt: z.string(),
  user: userProfileSchema,
})

export const messageSchema = z.object({
  message: z.string(),
})

export const checkoutSessionResultSchema = z
  .object({
    orderId: z.string(),
    orderNumber: z.string(),
    sessionId: z.string(),
    url: z.string().optional(),
    clientSecret: z.string().optional(),
    accessToken: z.string().optional(),
  })
  .refine((data) => Boolean(data.url || data.clientSecret), {
    message: 'checkout session must include url or clientSecret',
  })

const errorBodySchema = z.object({
  code: z.string(),
  message: z.string(),
})

export function parseApiEnvelope<T extends z.ZodType>(dataSchema: T, body: unknown): z.infer<T> {
  const envelopeSchema = z.object({
    data: dataSchema.nullable(),
    error: errorBodySchema.nullable(),
  })
  const envelope = envelopeSchema.parse(body) as {
    data: z.infer<T> | null
    error: { code: string; message: string } | null
  }
  if (envelope.error) {
    throw new ApiResponseError(envelope.error.message, envelope.error.code)
  }
  if (envelope.data === null) {
    throw new ApiResponseError('empty response data', 'EMPTY_RESPONSE')
  }
  return envelope.data
}

export class ApiResponseError extends Error {
  readonly code: string

  constructor(message: string, code: string) {
    super(message)
    this.name = 'ApiResponseError'
    this.code = code
  }
}
