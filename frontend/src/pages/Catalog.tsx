import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listProducts, Product } from '../api/client'

function formatPrice(cents: number, currency: string) {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(cents / 100)
}

export default function Catalog() {
  const [products, setProducts] = useState<Product[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    listProducts()
      .then(setProducts)
      .catch((e) => setError(e instanceof Error ? e.message : 'Failed to load products'))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <p>Loading products...</p>
  if (error) return <p className="error">{error}</p>

  return (
    <div>
      <h1>Stripe Payment Prototype</h1>
      <p>Choose a product and checkout via Hosted (redirect) or Embedded (on-site).</p>
      {products.map((p) => (
        <div key={p.id} className="card">
          <h2>{p.name}</h2>
          {p.description && <p>{p.description}</p>}
          <p><strong>{formatPrice(p.unitAmountCents, p.currency)}</strong></p>
          <div className="actions">
            <Link className="btn" to={`/checkout/hosted?productId=${p.id}`}>
              Pay (Hosted)
            </Link>
            <Link className="btn secondary" to={`/checkout/embedded?productId=${p.id}`}>
              Pay (Embedded)
            </Link>
          </div>
        </div>
      ))}
    </div>
  )
}
