# ============================================================================
# Arsenale — Deployment via Ansible
# ============================================================================
# Usage:
#   make setup      — First-time setup: install Ansible collections, generate vault + certs
#   make dev        — Start development infrastructure (postgres + gocache)
#   make deploy     — Deploy full production stack
#   make help       — Show all available targets
# ============================================================================

SHELL := /bin/bash
ANSIBLE_DIR := deployment/ansible
PLAYBOOK := cd $(ANSIBLE_DIR) && ansible-playbook
VAULT_FLAG := --ask-vault-pass

.DEFAULT_GOAL := help

# ── Dependency check ────────────────────────────────────────────────────────

.PHONY: _check-ansible
_check-ansible:
	@command -v ansible-playbook >/dev/null 2>&1 || { \
		printf "\033[1;31mERROR: Ansible is not installed.\033[0m\n\n"; \
		printf "Install it with one of:\n"; \
		printf "  pip install ansible          # Any platform (recommended)\n"; \
		printf "  pipx install ansible          # Isolated install\n"; \
		printf "  brew install ansible          # macOS (Homebrew)\n"; \
		printf "  sudo dnf install ansible-core # Fedora / RHEL\n"; \
		printf "  sudo apt install ansible      # Debian / Ubuntu\n"; \
		printf "  sudo pacman -S ansible        # Arch Linux\n"; \
		printf "\nThen run: make setup\n"; \
		exit 1; \
	}

# ── First-time setup ───────────────────────────────────────────────────────

.PHONY: setup
setup: _check-ansible  ## First-time setup: install collections, generate vault + certs
	cd $(ANSIBLE_DIR) && ansible-galaxy collection install -r requirements.yml 2>/dev/null || true
	@if [ ! -f $(ANSIBLE_DIR)/inventory/group_vars/all/vault.yml ]; then \
		echo "Generating Ansible Vault..."; \
		cd $(ANSIBLE_DIR) && ./scripts/generate-vault.sh; \
	else \
		echo "Vault already exists. To regenerate: make vault"; \
	fi
	@echo ""
	@echo "Setup complete. Next steps:"
	@echo "  make dev       — Start development environment"
	@echo "  make deploy    — Deploy production stack"

# ── Development ─────────────────────────────────────────────────────────────

.PHONY: dev
dev: _check-ansible  ## Start dev infrastructure (postgres + gocache) and generate .env
	$(PLAYBOOK) playbooks/dev.yml $(VAULT_FLAG)

.PHONY: dev-down
dev-down: _check-ansible  ## Stop dev infrastructure
	$(PLAYBOOK) playbooks/dev.yml $(VAULT_FLAG) -e arsenale_dev_state=absent

# ── Production ──────────────────────────────────────────────────────────────

.PHONY: deploy
deploy: _check-ansible  ## Deploy full production stack
	$(PLAYBOOK) playbooks/deploy.yml $(VAULT_FLAG)

# ── Operations ──────────────────────────────────────────────────────────────

.PHONY: status
status: _check-ansible  ## Show service status
	$(PLAYBOOK) playbooks/deploy.yml $(VAULT_FLAG) --tags status

.PHONY: logs
logs:  ## Follow service logs (pass SVC= for specific service)
	podman compose -f $$(find /opt/arsenale -name docker-compose.yml 2>/dev/null || echo "docker-compose.yml") logs -f $(SVC)

.PHONY: backup
backup: _check-ansible  ## Create database backup
	$(PLAYBOOK) playbooks/backup.yml $(VAULT_FLAG)

.PHONY: rotate
rotate: _check-ansible  ## Rotate system secrets
	$(PLAYBOOK) playbooks/rotate-secrets.yml $(VAULT_FLAG)

# ── Secrets & Certificates ──────────────────────────────────────────────────

.PHONY: vault
vault: _check-ansible  ## Generate or edit Ansible Vault
	@if [ -f $(ANSIBLE_DIR)/inventory/group_vars/all/vault.yml ]; then \
		ansible-vault edit $(ANSIBLE_DIR)/inventory/group_vars/all/vault.yml; \
	else \
		cd $(ANSIBLE_DIR) && ./scripts/generate-vault.sh; \
	fi

.PHONY: certs
certs: _check-ansible  ## Regenerate TLS certificates
	$(PLAYBOOK) playbooks/deploy.yml $(VAULT_FLAG) --tags certificates

# ── Cleanup ─────────────────────────────────────────────────────────────────

.PHONY: clean
clean: _check-ansible  ## Stop and remove all containers and volumes
	$(PLAYBOOK) playbooks/deploy.yml $(VAULT_FLAG) -e arsenale_state=absent

# ── Help ────────────────────────────────────────────────────────────────────

.PHONY: help
help:  ## Show available targets
	@printf "\033[1mArsenale Deployment\033[0m\n\n"
	@printf "Prerequisites: ansible (make setup will guide you)\n\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@printf "\n"
