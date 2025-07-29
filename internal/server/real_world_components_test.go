package server

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealWorldTemplComponents validates hot reload with realistic templ components
func TestRealWorldTemplComponents(t *testing.T) {
	t.Run("blog_post_component", func(t *testing.T) {
		tempDir := fmt.Sprintf("blog_test_%d", time.Now().UnixNano())
		require.NoError(t, os.MkdirAll(tempDir, 0755))
		defer os.RemoveAll(tempDir)

		reg := registry.NewComponentRegistry()
		scanner := scanner.NewComponentScanner(reg)

		// Create realistic blog post component
		blogComponent := `package components

import "time"

type BlogPost struct {
	Title    string
	Content  string
	Author   string
	Date     time.Time
	Tags     []string
}

templ BlogPostCard(post BlogPost) {
	<article class="blog-post-card">
		<header>
			<h2 class="post-title">{post.Title}</h2>
			<div class="post-meta">
				<span class="author">By {post.Author}</span>
				<time class="date" datetime={post.Date.Format("2006-01-02")}>
					{post.Date.Format("January 2, 2006")}
				</time>
			</div>
		</header>
		<div class="post-content">
			{post.Content}
		</div>
		<footer class="post-tags">
			for _, tag := range post.Tags {
				<span class="tag">#{tag}</span>
			}
		</footer>
	</article>
}`

		componentPath := filepath.Join(tempDir, "blog_post.templ")
		require.NoError(t, os.WriteFile(componentPath, []byte(blogComponent), 0644))

		// Scan component
		err := scanner.ScanFile(componentPath)
		assert.NoError(t, err, "Blog component should scan successfully")

		// Verify component registration
		component, exists := reg.Get("BlogPostCard")
		require.True(t, exists, "BlogPostCard should be registered")
		
		assert.Equal(t, "BlogPostCard", component.Name)
		assert.Equal(t, 1, len(component.Parameters), "Should have 1 parameter")
		assert.Equal(t, "post", component.Parameters[0].Name)
		assert.Equal(t, "BlogPost", component.Parameters[0].Type)
		assert.Contains(t, component.Imports, "time", "Should detect time import")

		t.Log("✅ Real-world blog component validated successfully")
	})

	t.Run("navigation_component", func(t *testing.T) {
		tempDir := fmt.Sprintf("nav_test_%d", time.Now().UnixNano())
		require.NoError(t, os.MkdirAll(tempDir, 0755))
		defer os.RemoveAll(tempDir)

		reg := registry.NewComponentRegistry()
		scanner := scanner.NewComponentScanner(reg)

		// Create realistic navigation component
		navComponent := `package components

type NavItem struct {
	Label string
	URL   string
	Active bool
	Icon  string
}

type NavigationProps struct {
	Brand     string
	Items     []NavItem
	UserName  string
	LoggedIn  bool
}

templ Navigation(props NavigationProps) {
	<nav class="navbar" role="navigation" aria-label="main navigation">
		<div class="navbar-brand">
			<a class="navbar-item" href="/">
				if props.Brand != "" {
					{props.Brand}
				} else {
					MyApp
				}
			</a>
		</div>
		
		<div class="navbar-menu">
			<div class="navbar-start">
				for _, item := range props.Items {
					<a 
						class={
							"navbar-item", 
							templ.KV("is-active", item.Active),
						}
						href={templ.URL(item.URL)}
					>
						if item.Icon != "" {
							<i class={fmt.Sprintf("icon-%s", item.Icon)}></i>
						}
						{item.Label}
					</a>
				}
			</div>
			
			<div class="navbar-end">
				if props.LoggedIn {
					<div class="navbar-item has-dropdown is-hoverable">
						<a class="navbar-link">
							{props.UserName}
						</a>
						<div class="navbar-dropdown">
							<a class="navbar-item" href="/profile">Profile</a>
							<a class="navbar-item" href="/settings">Settings</a>
							<hr class="navbar-divider"/>
							<a class="navbar-item" href="/logout">Logout</a>
						</div>
					</div>
				} else {
					<div class="navbar-item">
						<div class="buttons">
							<a class="button is-primary" href="/signup">Sign up</a>
							<a class="button is-light" href="/login">Log in</a>
						</div>
					</div>
				}
			</div>
		</div>
	</nav>
}`

		componentPath := filepath.Join(tempDir, "navigation.templ")
		require.NoError(t, os.WriteFile(componentPath, []byte(navComponent), 0644))

		// Scan component
		err := scanner.ScanFile(componentPath)
		assert.NoError(t, err, "Navigation component should scan successfully")

		// Verify component registration
		component, exists := reg.Get("Navigation")
		require.True(t, exists, "Navigation should be registered")
		
		assert.Equal(t, "Navigation", component.Name)
		assert.Equal(t, 1, len(component.Parameters), "Should have 1 parameter")
		assert.Equal(t, "props", component.Parameters[0].Name)
		assert.Equal(t, "NavigationProps", component.Parameters[0].Type)

		t.Log("✅ Real-world navigation component validated successfully")
	})

	t.Run("form_component", func(t *testing.T) {
		tempDir := fmt.Sprintf("form_test_%d", time.Now().UnixNano())
		require.NoError(t, os.MkdirAll(tempDir, 0755))
		defer os.RemoveAll(tempDir)

		reg := registry.NewComponentRegistry()
		scanner := scanner.NewComponentScanner(reg)

		// Create realistic form component
		formComponent := `package components

type ValidationError struct {
	Field   string
	Message string
}

type ContactFormData struct {
	Name    string
	Email   string
	Subject string
	Message string
	Errors  []ValidationError
}

templ ContactForm(data ContactFormData, csrf string) {
	<form class="contact-form" method="POST" action="/contact">
		<input type="hidden" name="_csrf" value={csrf}/>
		
		<div class="field">
			<label class="label" for="name">Name *</label>
			<div class="control">
				<input 
					class={
						"input",
						templ.KV("is-danger", hasFieldError("name", data.Errors)),
					}
					type="text" 
					id="name" 
					name="name" 
					value={data.Name}
					required
				/>
			</div>
			{renderFieldError("name", data.Errors)}
		</div>

		<div class="field">
			<label class="label" for="email">Email *</label>
			<div class="control">
				<input 
					class={
						"input",
						templ.KV("is-danger", hasFieldError("email", data.Errors)),
					}
					type="email" 
					id="email" 
					name="email" 
					value={data.Email}
					required
				/>
			</div>
			{renderFieldError("email", data.Errors)}
		</div>

		<div class="field">
			<label class="label" for="subject">Subject</label>
			<div class="control">
				<input 
					class="input"
					type="text" 
					id="subject" 
					name="subject" 
					value={data.Subject}
				/>
			</div>
		</div>

		<div class="field">
			<label class="label" for="message">Message *</label>
			<div class="control">
				<textarea 
					class={
						"textarea",
						templ.KV("is-danger", hasFieldError("message", data.Errors)),
					}
					id="message" 
					name="message" 
					rows="5"
					required
				>{data.Message}</textarea>
			</div>
			{renderFieldError("message", data.Errors)}
		</div>

		<div class="field is-grouped">
			<div class="control">
				<button class="button is-primary" type="submit">
					Send Message
				</button>
			</div>
			<div class="control">
				<button class="button is-light" type="button" onclick="resetForm()">
					Reset
				</button>
			</div>
		</div>
	</form>
}

templ renderFieldError(fieldName string, errors []ValidationError) {
	for _, err := range errors {
		if err.Field == fieldName {
			<p class="help is-danger">{err.Message}</p>
		}
	}
}

func hasFieldError(fieldName string, errors []ValidationError) bool {
	for _, err := range errors {
		if err.Field == fieldName {
			return true
		}
	}
	return false
}`

		componentPath := filepath.Join(tempDir, "contact_form.templ")
		require.NoError(t, os.WriteFile(componentPath, []byte(formComponent), 0644))

		// Scan component
		err := scanner.ScanFile(componentPath)
		assert.NoError(t, err, "Form component should scan successfully")

		// Verify multiple components registered
		contactForm, exists := reg.Get("ContactForm")
		require.True(t, exists, "ContactForm should be registered")
		assert.Equal(t, 2, len(contactForm.Parameters), "Should have 2 parameters")

		renderError, exists := reg.Get("renderFieldError")
		require.True(t, exists, "renderFieldError should be registered")
		assert.Equal(t, 2, len(renderError.Parameters), "Should have 2 parameters")

		t.Log("✅ Real-world form component validated successfully")
	})

	t.Run("dashboard_component", func(t *testing.T) {
		tempDir := fmt.Sprintf("dashboard_test_%d", time.Now().UnixNano())
		require.NoError(t, os.MkdirAll(tempDir, 0755))
		defer os.RemoveAll(tempDir)

		reg := registry.NewComponentRegistry()
		scanner := scanner.NewComponentScanner(reg)

		// Create realistic dashboard component
		dashboardComponent := `package components

import (
	"fmt"
	"time"
)

type Metric struct {
	Name   string
	Value  float64
	Unit   string
	Change float64 // Percentage change
	Icon   string
}

type ChartData struct {
	Labels []string
	Values []float64
}

type DashboardProps struct {
	Title       string
	Metrics     []Metric
	ChartData   ChartData
	LastUpdated time.Time
	RefreshRate int // seconds
}

templ Dashboard(props DashboardProps) {
	<div class="dashboard" data-refresh-rate={fmt.Sprintf("%d", props.RefreshRate)}>
		<header class="dashboard-header">
			<h1 class="title">{props.Title}</h1>
			<div class="last-updated">
				Last updated: 
				<time datetime={props.LastUpdated.Format(time.RFC3339)}>
					{props.LastUpdated.Format("15:04:05")}
				</time>
			</div>
		</header>

		<div class="metrics-grid">
			for _, metric := range props.Metrics {
				@MetricCard(metric)
			}
		</div>

		<div class="chart-section">
			<h2 class="section-title">Trends</h2>
			@LineChart(props.ChartData)
		</div>
	</div>
}

templ MetricCard(metric Metric) {
	<div class="metric-card">
		<div class="metric-icon">
			<i class={fmt.Sprintf("icon-%s", metric.Icon)}></i>
		</div>
		<div class="metric-content">
			<div class="metric-value">
				{fmt.Sprintf("%.2f", metric.Value)}
				if metric.Unit != "" {
					<span class="unit">{metric.Unit}</span>
				}
			</div>
			<div class="metric-name">{metric.Name}</div>
			if metric.Change != 0 {
				<div class={
					"metric-change",
					templ.KV("positive", metric.Change > 0),
					templ.KV("negative", metric.Change < 0),
				}>
					{fmt.Sprintf("%.1f%%", metric.Change)}
				</div>
			}
		</div>
	</div>
}

templ LineChart(data ChartData) {
	<div class="chart-container">
		<canvas 
			id="trend-chart" 
			data-labels={fmt.Sprintf("%v", data.Labels)}
			data-values={fmt.Sprintf("%v", data.Values)}
		></canvas>
	</div>
}`

		componentPath := filepath.Join(tempDir, "dashboard.templ")
		require.NoError(t, os.WriteFile(componentPath, []byte(dashboardComponent), 0644))

		// Scan component
		err := scanner.ScanFile(componentPath)
		assert.NoError(t, err, "Dashboard component should scan successfully")

		// Verify all components registered
		dashboard, exists := reg.Get("Dashboard")
		require.True(t, exists, "Dashboard should be registered")
		assert.Equal(t, 1, len(dashboard.Parameters), "Should have 1 parameter")

		metricCard, exists := reg.Get("MetricCard")
		require.True(t, exists, "MetricCard should be registered")
		assert.Equal(t, 1, len(metricCard.Parameters), "Should have 1 parameter")

		lineChart, exists := reg.Get("LineChart")
		require.True(t, exists, "LineChart should be registered")
		assert.Equal(t, 1, len(lineChart.Parameters), "Should have 1 parameter")

		// Verify imports detected
		assert.Contains(t, dashboard.Imports, "fmt", "Should detect fmt import")
		assert.Contains(t, dashboard.Imports, "time", "Should detect time import")

		t.Log("✅ Real-world dashboard component validated successfully")
	})

	t.Run("e_commerce_product", func(t *testing.T) {
		tempDir := fmt.Sprintf("product_test_%d", time.Now().UnixNano())
		require.NoError(t, os.MkdirAll(tempDir, 0755))
		defer os.RemoveAll(tempDir)

		reg := registry.NewComponentRegistry()
		scanner := scanner.NewComponentScanner(reg)

		// Create realistic e-commerce product component
		productComponent := `package components

import (
	"fmt"
	"strconv"
)

type Price struct {
	Amount   float64
	Currency string
	Sale     *float64 // Sale price if on sale
}

type Review struct {
	Rating  int
	Comment string
	Author  string
}

type Product struct {
	ID          string
	Name        string
	Description string
	Images      []string
	Price       Price
	InStock     bool
	StockCount  int
	Rating      float64
	ReviewCount int
	Reviews     []Review
	Categories  []string
	Variants    []ProductVariant
}

type ProductVariant struct {
	ID    string
	Name  string
	Value string
	Price *Price
}

templ ProductCard(product Product, showDetails bool) {
	<div class="product-card" data-product-id={product.ID}>
		<div class="product-image">
			if len(product.Images) > 0 {
				<img 
					src={product.Images[0]} 
					alt={product.Name}
					loading="lazy"
				/>
			} else {
				<div class="no-image">No Image</div>
			}
			if product.Price.Sale != nil {
				<div class="sale-badge">Sale</div>
			}
		</div>

		<div class="product-info">
			<h3 class="product-name">
				<a href={templ.URL(fmt.Sprintf("/products/%s", product.ID))}>
					{product.Name}
				</a>
			</h3>

			if showDetails {
				<p class="product-description">{product.Description}</p>
			}

			<div class="product-rating">
				@StarRating(product.Rating)
				<span class="review-count">
					({strconv.Itoa(product.ReviewCount)} reviews)
				</span>
			</div>

			<div class="product-price">
				if product.Price.Sale != nil {
					<span class="original-price">
						{fmt.Sprintf("%.2f %s", product.Price.Amount, product.Price.Currency)}
					</span>
					<span class="sale-price">
						{fmt.Sprintf("%.2f %s", *product.Price.Sale, product.Price.Currency)}
					</span>
				} else {
					<span class="current-price">
						{fmt.Sprintf("%.2f %s", product.Price.Amount, product.Price.Currency)}
					</span>
				}
			</div>

			<div class="product-stock">
				if product.InStock {
					if product.StockCount <= 5 {
						<span class="low-stock">Only {strconv.Itoa(product.StockCount)} left!</span>
					} else {
						<span class="in-stock">In Stock</span>
					}
				} else {
					<span class="out-of-stock">Out of Stock</span>
				}
			</div>

			if len(product.Variants) > 0 {
				<div class="product-variants">
					for _, variant := range product.Variants {
						<div class="variant">
							<strong>{variant.Name}:</strong> {variant.Value}
						</div>
					}
				</div>
			}

			<div class="product-actions">
				<button 
					class="btn btn-primary add-to-cart"
					data-product-id={product.ID}
					disabled?={!product.InStock}
				>
					if product.InStock {
						Add to Cart
					} else {
						Notify When Available
					}
				</button>
				<button class="btn btn-secondary wishlist" data-product-id={product.ID}>
					♡ Wishlist
				</button>
			</div>
		</div>
	</div>
}

templ StarRating(rating float64) {
	<div class="star-rating" data-rating={fmt.Sprintf("%.1f", rating)}>
		for i := 1; i <= 5; i++ {
			if float64(i) <= rating {
				<span class="star filled">★</span>
			} else if float64(i)-0.5 <= rating {
				<span class="star half">★</span>
			} else {
				<span class="star empty">☆</span>
			}
		}
	</div>
}`

		componentPath := filepath.Join(tempDir, "product.templ")
		require.NoError(t, os.WriteFile(componentPath, []byte(productComponent), 0644))

		// Scan component
		err := scanner.ScanFile(componentPath)
		assert.NoError(t, err, "Product component should scan successfully")

		// Verify components registered
		productCard, exists := reg.Get("ProductCard")
		require.True(t, exists, "ProductCard should be registered")
		assert.Equal(t, 2, len(productCard.Parameters), "Should have 2 parameters")

		starRating, exists := reg.Get("StarRating")
		require.True(t, exists, "StarRating should be registered")
		assert.Equal(t, 1, len(starRating.Parameters), "Should have 1 parameter")

		// Verify imports
		assert.Contains(t, productCard.Imports, "fmt", "Should detect fmt import")
		assert.Contains(t, productCard.Imports, "strconv", "Should detect strconv import")

		t.Log("✅ Real-world e-commerce component validated successfully")
	})
}

// TestRealWorldTemplModification tests hot reload with realistic component modifications
func TestRealWorldTemplModification(t *testing.T) {
	tempDir := fmt.Sprintf("modification_test_%d", time.Now().UnixNano())
	require.NoError(t, os.MkdirAll(tempDir, 0755))
	defer os.RemoveAll(tempDir)

	reg := registry.NewComponentRegistry()
	scanner := scanner.NewComponentScanner(reg)

	// Start with simple button component
	originalComponent := `package components

templ Button(text string, variant string) {
	<button class={fmt.Sprintf("btn btn-%s", variant)}>
		{text}
	</button>
}`

	componentPath := filepath.Join(tempDir, "button.templ")
	require.NoError(t, os.WriteFile(componentPath, []byte(originalComponent), 0644))

	// Initial scan
	err := scanner.ScanFile(componentPath)
	assert.NoError(t, err, "Initial component should scan successfully")

	button, exists := reg.Get("Button")
	require.True(t, exists, "Button should be registered")
	assert.Equal(t, 2, len(button.Parameters), "Should have 2 parameters initially")

	// Modify to add more functionality
	enhancedComponent := `package components

import "fmt"

type ButtonProps struct {
	Text     string
	Variant  string
	Size     string
	Disabled bool
	Icon     string
	OnClick  string
}

templ Button(props ButtonProps) {
	<button 
		class={
			fmt.Sprintf("btn btn-%s btn-%s", props.Variant, props.Size),
			templ.KV("disabled", props.Disabled),
		}
		disabled?={props.Disabled}
		onclick={props.OnClick}
	>
		if props.Icon != "" {
			<i class={fmt.Sprintf("icon-%s", props.Icon)}></i>
		}
		{props.Text}
	</button>
}

templ IconButton(icon string, variant string, size string) {
	@Button(ButtonProps{
		Text:    "",
		Variant: variant,
		Size:    size,
		Icon:    icon,
	})
}

templ LinkButton(text string, href string, variant string) {
	<a 
		href={templ.URL(href)}
		class={fmt.Sprintf("btn btn-%s", variant)}
		role="button"
	>
		{text}
	</a>
}`

	require.NoError(t, os.WriteFile(componentPath, []byte(enhancedComponent), 0644))

	// Re-scan after modification
	err = scanner.ScanFile(componentPath)
	assert.NoError(t, err, "Enhanced component should scan successfully")

	// Verify updated Button component
	updatedButton, exists := reg.Get("Button")
	require.True(t, exists, "Updated Button should be registered")
	assert.Equal(t, 1, len(updatedButton.Parameters), "Should have 1 parameter after update")
	assert.Equal(t, "props", updatedButton.Parameters[0].Name)
	assert.Equal(t, "ButtonProps", updatedButton.Parameters[0].Type)

	// Verify new components
	iconButton, exists := reg.Get("IconButton")
	require.True(t, exists, "IconButton should be registered")
	assert.Equal(t, 3, len(iconButton.Parameters), "Should have 3 parameters")

	linkButton, exists := reg.Get("LinkButton")
	require.True(t, exists, "LinkButton should be registered")
	assert.Equal(t, 3, len(linkButton.Parameters), "Should have 3 parameters")

	// Verify imports detected
	assert.Contains(t, updatedButton.Imports, "fmt", "Should detect fmt import")

	t.Log("✅ Real-world component modification validated successfully")
}