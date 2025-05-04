.PHONY: docs docs-test docs-clean

# Generate documentation
docs:
	@echo "==> Generating documentation..."
	go generate ./...

# Test documentation
docs-test:
	@echo "==> Testing documentation..."
	@tfplugindocs validate

# Clean documentation
docs-clean:
	@echo "==> Cleaning documentation..."
	@rm -rf docs/ 