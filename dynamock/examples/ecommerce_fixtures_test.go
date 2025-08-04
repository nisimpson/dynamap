package examples

import (
	"time"

	"github.com/nisimpson/dynamap"
	"github.com/nisimpson/dynamap/dynamock"
)

// ProductBuilder provides a fluent API for building test products.
type ProductBuilder struct {
	id       string
	category string
	price    int
	name     string
	created  time.Time
}

// NewProduct creates a new product builder.
func NewProduct() *ProductBuilder {
	return &ProductBuilder{
		created: time.Now(),
	}
}

// WithID sets the product ID.
func (b *ProductBuilder) WithID(id string) *ProductBuilder {
	b.id = id
	return b
}

// WithCategory sets the product category and uses it as the ref sort key.
func (b *ProductBuilder) WithCategory(category string) *ProductBuilder {
	b.category = category
	return b
}

// WithPrice sets the product price.
func (b *ProductBuilder) WithPrice(price int) *ProductBuilder {
	b.price = price
	return b
}

// WithName sets the product name.
func (b *ProductBuilder) WithName(name string) *ProductBuilder {
	b.name = name
	return b
}

// Build creates a TestEntity configured as a product.
func (b *ProductBuilder) Build() *dynamock.TestEntity {
	// Set the data to include product-specific fields
	data := map[string]interface{}{
		"id":       b.id,
		"category": b.category,
		"price":    b.price,
		"name":     b.name,
	}

	return dynamock.NewEntity(
		dynamock.WithID(b.id),
		dynamock.WithPrefix("product"),
		dynamock.WithLabel("product"),
		dynamock.WithRefSortKey(b.category),
		dynamock.WithCreated(b.created),
		dynamock.WithData(data),
	).Build()
}

// OrderBuilder provides a fluent API for building test orders.
type OrderBuilder struct {
	id         string
	customerID string
	products   []dynamap.Marshaler
	created    time.Time
}

// NewOrder creates a new order builder.
func NewOrder() *OrderBuilder {
	return &OrderBuilder{
		products: make([]dynamap.Marshaler, 0),
		created:  time.Now(),
	}
}

// WithID sets the order ID.
func (b *OrderBuilder) WithID(id string) *OrderBuilder {
	b.id = id
	return b
}

// WithCustomerID sets the customer ID.
func (b *OrderBuilder) WithCustomerID(customerID string) *OrderBuilder {
	b.customerID = customerID
	return b
}

// WithProduct adds a product to the order.
func (b *OrderBuilder) WithProduct(product dynamap.Marshaler) *OrderBuilder {
	b.products = append(b.products, product)
	return b
}

// WithProducts adds multiple products to the order.
func (b *OrderBuilder) WithProducts(products ...dynamap.Marshaler) *OrderBuilder {
	b.products = append(b.products, products...)
	return b
}

// Build creates a TestEntity configured as an order with product relationships.
func (b *OrderBuilder) Build() *dynamock.TestEntity {
	// Set the data to include order-specific fields
	data := map[string]interface{}{
		"id":          b.id,
		"customer_id": b.customerID,
	}

	options := []dynamock.EntityOption{
		dynamock.WithID(b.id),
		dynamock.WithPrefix("order"),
		dynamock.WithLabel("order"),
		dynamock.WithRefSortKey(b.created.Format("2006-01-02")),
		dynamock.WithCreated(b.created),
		dynamock.WithData(data),
	}

	// Add product relationships if any
	if len(b.products) > 0 {
		options = append(options, dynamock.WithRelationships("products", b.products...))
	}

	return dynamock.NewEntity(options...).Build()
}

// CustomerBuilder provides a fluent API for building test customers.
type CustomerBuilder struct {
	id      string
	email   string
	name    string
	tier    string
	created time.Time
}

// NewCustomer creates a new customer builder.
func NewCustomer() *CustomerBuilder {
	return &CustomerBuilder{
		created: time.Now(),
	}
}

// WithID sets the customer ID.
func (b *CustomerBuilder) WithID(id string) *CustomerBuilder {
	b.id = id
	return b
}

// WithEmail sets the customer email.
func (b *CustomerBuilder) WithEmail(email string) *CustomerBuilder {
	b.email = email
	return b
}

// WithName sets the customer name.
func (b *CustomerBuilder) WithName(name string) *CustomerBuilder {
	b.name = name
	return b
}

// WithTier sets the customer tier and uses it as the ref sort key.
func (b *CustomerBuilder) WithTier(tier string) *CustomerBuilder {
	b.tier = tier
	return b
}

// Premium sets the customer as premium tier.
func (b *CustomerBuilder) Premium() *CustomerBuilder {
	return b.WithTier("premium")
}

// Standard sets the customer as standard tier.
func (b *CustomerBuilder) Standard() *CustomerBuilder {
	return b.WithTier("standard")
}

// Build creates a TestEntity configured as a customer.
func (b *CustomerBuilder) Build() *dynamock.TestEntity {
	// Set the data to include customer-specific fields
	data := map[string]interface{}{
		"id":    b.id,
		"email": b.email,
		"name":  b.name,
		"tier":  b.tier,
	}

	return dynamock.NewEntity(
		dynamock.WithID(b.id),
		dynamock.WithPrefix("customer"),
		dynamock.WithLabel("customer"),
		dynamock.WithRefSortKey(b.tier),
		dynamock.WithCreated(b.created),
		dynamock.WithData(data),
	).Build()
}

// Quick Helper Functions

// QuickProduct creates a simple product entity with minimal configuration.
func QuickProduct(id, category string) *dynamock.TestEntity {
	return NewProduct().
		WithID(id).
		WithCategory(category).
		Build()
}

// QuickOrder creates a simple order entity with minimal configuration.
func QuickOrder(id, customerID string) *dynamock.TestEntity {
	return NewOrder().
		WithID(id).
		WithCustomerID(customerID).
		Build()
}

// QuickCustomer creates a simple customer entity with minimal configuration.
func QuickCustomer(id, email string) *dynamock.TestEntity {
	return NewCustomer().
		WithID(id).
		WithEmail(email).
		Standard().
		Build()
}

// QuickPremiumCustomer creates a premium customer entity.
func QuickPremiumCustomer(id, email string) *dynamock.TestEntity {
	return NewCustomer().
		WithID(id).
		WithEmail(email).
		Premium().
		Build()
}
