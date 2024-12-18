resource "stripe_coupon" "example" {
  name = "Example coupon"
  applies_to = [
    "price_...",
  ]
  metadata = {
    foo = "bar"
  }
}
