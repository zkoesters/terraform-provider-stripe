resource "stripe_product" "example" {
  name = "Example product"
  metadata = {
    foo = "bar"
  }
}
