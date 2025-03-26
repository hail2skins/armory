package data

// EnhanceHomePageSEO adds standard SEO metadata for the home page
func EnhanceHomePageSEO(authData AuthData, host string) AuthData {
	authData = authData.WithMetaDescription("The Virtual Armory - Your digital home for tracking your home arsenal safely and securely. Keep detailed records of all your firearms in one secure location.")
	authData = authData.WithOgImage("/assets/tva-logo.jpg")
	authData = authData.WithCanonicalURL("https://" + host)
	authData = authData.WithMetaKeywords("virtual armory, gun collection, firearm tracking, arsenal management, gun inventory, firearm collection")
	authData = authData.WithOgType("website")

	// Add JSON-LD structured data for the website
	authData = authData.WithStructuredData(map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "WebSite",
		"name":        "The Virtual Armory",
		"url":         "https://" + host,
		"description": "Your digital home for tracking your home arsenal safely and securely.",
		"potentialAction": map[string]interface{}{
			"@type":       "SearchAction",
			"target":      "https://" + host + "/search?q={search_term_string}",
			"query-input": "required name=search_term_string",
		},
	})

	return authData
}

// EnhanceAboutPageSEO adds standard SEO metadata for the about page
func EnhanceAboutPageSEO(authData AuthData, host string) AuthData {
	authData = authData.WithMetaDescription("Learn about The Virtual Armory and our mission to provide firearm enthusiasts with a secure, private, and comprehensive platform to track their collections.")
	authData = authData.WithOgImage("/assets/virtualarmory.jpg")
	authData = authData.WithCanonicalURL("https://" + host + "/about")
	authData = authData.WithMetaKeywords("about virtual armory, firearm tracking mission, gun collection platform, secure gun inventory")
	authData = authData.WithOgType("website")

	// Add JSON-LD structured data for the about page
	authData = authData.WithStructuredData(map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "AboutPage",
		"name":        "About The Virtual Armory",
		"description": "Learn about The Virtual Armory and our mission to provide firearm enthusiasts with a secure, private, and comprehensive platform to track their collections.",
	})

	return authData
}

// EnhanceContactPageSEO adds standard SEO metadata for the contact page
func EnhanceContactPageSEO(authData AuthData, host string) AuthData {
	authData = authData.WithMetaDescription("Contact The Virtual Armory with your questions, feedback, or feature suggestions. We're here to help you get the most out of your digital firearm collection management.")
	authData = authData.WithOgImage("/assets/contact-bench.jpg")
	authData = authData.WithCanonicalURL("https://" + host + "/contact")
	authData = authData.WithMetaKeywords("contact virtual armory, firearm tracking help, gun collection support, feedback")
	authData = authData.WithOgType("website")

	// Add JSON-LD structured data for the contact page
	authData = authData.WithStructuredData(map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "ContactPage",
		"name":        "Contact The Virtual Armory",
		"description": "Contact The Virtual Armory with your questions, feedback, or feature suggestions.",
	})

	return authData
}

// EnhancePricingPageSEO adds standard SEO metadata for the pricing page
func EnhancePricingPageSEO(authData AuthData, host string) AuthData {
	authData = authData.WithMetaDescription("Choose from our flexible pricing plans for The Virtual Armory. From free basic access to premium lifetime memberships, find the plan that fits your collection management needs.")
	authData = authData.WithOgImage("/assets/pricing-back.jpg")
	authData = authData.WithCanonicalURL("https://" + host + "/pricing")
	authData = authData.WithMetaKeywords("virtual armory pricing, gun collection subscription, firearm tracking plans, arsenal management cost")
	authData = authData.WithOgType("website")

	// Add JSON-LD structured data for the pricing page
	authData = authData.WithStructuredData(map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "Product",
		"name":        "The Virtual Armory Subscription",
		"description": "Digital firearm collection management service for responsible gun owners",
		"offers": map[string]interface{}{
			"@type":         "AggregateOffer",
			"priceCurrency": "USD",
			"lowPrice":      "0",
			"highPrice":     "100",
			"offerCount":    "4",
		},
	})

	return authData
}
