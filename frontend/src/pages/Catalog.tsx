import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listProducts } from '../api/client'
import { ErrorMessage } from '../components/ErrorMessage'
import { LoadingMessage } from '../components/LoadingMessage'
import { formatPrice } from '../lib/formatPrice'
import type { Product } from '../types/api'

export function Catalog() {
  const [products, setProducts] = useState<Product[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false

    listProducts()
      .then((nextProducts) => {
        if (!cancelled) {
          setProducts(nextProducts)
        }
      })
      .catch((loadError) => {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : 'Failed to load products')
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [])

  if (loading) {
    return <LoadingMessage message="Loading products..." />
  }
  if (error) {
    return <ErrorMessage message={error} />
  }

  return (
    <div>
      <h1>Stripe Payment Prototype</h1>
      <p>
        Choose a product and checkout via Hosted (redirect) or Embedded (on-site).{' '}
        <Link to="/orders">View my orders</Link>
      </p>
      {products.map((product) => (
        <div key={product.id} className="card">
          <h2>{product.name}</h2>
          {product.description && <p>{product.description}</p>}
          <p>
            <strong>{formatPrice(product.unitAmountCents, product.currency)}</strong>
          </p>
          <div className="actions">
            <Link className="btn" to={`/checkout/hosted?productId=${product.id}`}>
              Pay (Hosted)
            </Link>
            <Link className="btn secondary" to={`/checkout/embedded?productId=${product.id}`}>
              Pay (Embedded)
            </Link>
          </div>
        </div>
      ))}
    </div>
  )
}
