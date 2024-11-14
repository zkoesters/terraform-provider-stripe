// Copyright (c) 2024 Zachary Koesters
// SPDX-License-Identifier: MPL-2.0

package provider

//func TestAccPriceResource(t *testing.T) {
//	resource.Test(t, resource.TestCase{
//		PreCheck:                 func() { testAccPreCheck(t) },
//		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
//		Steps: []resource.TestStep{
//			// Create and Read testing
//			{
//				Config: testAccCouponResourceConfigCreate,
//				Check: resource.ComposeAggregateTestCheckFunc(
//					resource.TestCheckResourceAttr("stripe_coupon.test", "amount_off", "1000"),
//					resource.TestCheckResourceAttr("stripe_coupon.test", "name", "test"),
//					resource.TestCheckResourceAttr("stripe_coupon.test", "currency", "usd"),
//					resource.TestCheckResourceAttr("stripe_coupon.test", "duration", "once"),
//				),
//			},
//			// ImportState testing
//			{
//				ResourceName:      "stripe_coupon.test",
//				ImportState:       true,
//				ImportStateVerify: false,
//			},
//			// Update and Read testing
//			{
//				Config: testAccCouponResourceConfigUpdate,
//				Check: resource.ComposeAggregateTestCheckFunc(
//					resource.TestCheckResourceAttr("stripe_coupon.test", "amount_off", "1000"),
//					resource.TestCheckResourceAttr("stripe_coupon.test", "name", "test_updated"),
//					resource.TestCheckResourceAttr("stripe_coupon.test", "currency", "usd"),
//					resource.TestCheckResourceAttr("stripe_coupon.test", "duration", "once"),
//				),
//			},
//			// Replace and Read testing
//			{
//				Config: testAccCouponResourceConfigReplace,
//				Check: resource.ComposeAggregateTestCheckFunc(
//					resource.TestCheckResourceAttr("stripe_coupon.test", "amount_off", "2000"),
//					resource.TestCheckResourceAttr("stripe_coupon.test", "name", "test_updated_again"),
//					resource.TestCheckResourceAttr("stripe_coupon.test", "currency", "usd"),
//					resource.TestCheckResourceAttr("stripe_coupon.test", "duration", "once"),
//				),
//			},
//			// Delete testing automatically occurs in TestCase
//		},
//	})
//}
//
//const (
//	testAccCouponResourceConfigCreate string = `
//resource "stripe_coupon" "test" {
//  name = "test"
//  amount_off = 1000
//  currency = "usd"
//  duration = "once"
//  metadata = {
//	test = "test"
//  }
//}
//`
//	testAccCouponResourceConfigUpdate string = `
//resource "stripe_coupon" "test" {
//  name = "test_updated"
//  amount_off = 1000
//  currency = "usd"
//  duration = "once"
//  metadata = {
//	test = "test"
//  }
//}
//`
//	testAccCouponResourceConfigReplace string = `
//resource "stripe_coupon" "test" {
//  name = "test_updated_again"
//  amount_off = 2000
//  currency = "usd"
//  duration = "once"
//  metadata = {
//	test = "test"
//  }
//}
//`
//)
