#!/usr/bin/env bash
#
# KorisPanel — unified installer + management CLI (single file)
#   Run without arguments: interactive menu (install if absent, else manage).
#   Subcommands:
#     install        Install / repair the Docker deployment (prompts for SSL)
#     start stop restart status logs follow update
#     config uninstall reinstall downgrade clean db pgadmin
#     enable disable node-status node-restart node-logs help
#

# ─── Colors ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# ─── Logging ─────────────────────────────────────────────────────────────────

log() {
  echo -e "${GREEN}[+]${NC} $*"
}

warn() {
  echo -e "${YELLOW}[!]${NC} $*"
}

err() {
  echo -e "${RED}[✗]${NC} $*" >&2
  exit 1
}

# ─── Cryptographic Utilities ─────────────────────────────────────────────────

# Generate a cryptographically random hex string.
# Usage: gen_secret [bytes]   (default: 32 bytes → 64 hex chars)
gen_secret() {
  local length="${1:-32}"
  openssl rand -hex "${length}" 2>/dev/null \
    || head -c "${length}" /dev/urandom | od -An -tx1 | tr -d ' \n'
}

# ─── OS Detection ────────────────────────────────────────────────────────────

# Validate that the host OS is a supported distribution.
# Exits with error if unsupported.
detect_os() {
  [[ -f /etc/os-release ]] || err "Unsupported OS: /etc/os-release not found"
  local os_id os_version
  os_id=$(. /etc/os-release && echo "$ID")
  os_version=$(. /etc/os-release && echo "$VERSION_ID")
  case "${os_id}" in
    ubuntu)
      if [[ "${os_version%%.*}" -lt 22 ]]; then
        err "Unsupported: ${os_id} ${os_version}. Need Ubuntu 22.04+"
      fi
      ;;
    debian)
      if [[ "${os_version%%.*}" -lt 12 ]]; then
        err "Unsupported: ${os_id} ${os_version}. Need Debian 12+"
      fi
      ;;
    *)
      err "Unsupported: ${os_id} ${os_version}. Need Ubuntu 22.04+ or Debian 12+"
      ;;
  esac
  log "Detected ${os_id} ${os_version}"
}

# ─── Precondition Checks ─────────────────────────────────────────────────────

# Require the script is running as root (EUID 0).
require_root() {
  [[ "$(id -u)" -eq 0 ]] || err "Must run as root"
}

# Require Docker daemon is installed and reachable.
require_docker() {
  command -v docker &>/dev/null || err "Docker is not installed"
  docker info &>/dev/null || err "Docker daemon not available"
}

# Require a specific Docker container is running.
# Usage: require_container <name>
require_container() {
  local name="${1:?require_container: container name required}"
  require_docker
  local state
  state=$(docker inspect -f '{{.State.Status}}' "${name}" 2>/dev/null) || true
  if [[ "${state}" != "running" ]]; then
    err "Container '${name}' is not running (state: ${state:-not found})"
  fi
}

# ─── User Interaction ─────────────────────────────────────────────────────────

# Prompt the user for confirmation. Aborts (exit 1) if not "yes".
# Usage: confirm "Are you sure you want to do X?"
confirm() {
  local msg="${1:-Are you sure?}"
  echo -e "${YELLOW}${msg}${NC}"
  read -rp "Type 'yes' to confirm: " answer </dev/tty
  if [[ "${answer}" != "yes" ]]; then
    echo "Operation cancelled."
    exit 1
  fi
}

# ─── Formatting ──────────────────────────────────────────────────────────────

# Format a byte count into the largest whole-number unit (B, KB, MB, GB).
# Uses 1024-based units.
# Usage: human_bytes 1048576  → "1 MB"
human_bytes() {
  local bytes="${1:?human_bytes: byte count required}"

  # Validate input is a non-negative integer
  if ! [[ "${bytes}" =~ ^[0-9]+$ ]]; then
    echo "0 B"
    return
  fi

  if (( bytes >= 1073741824 )); then
    echo "$(( bytes / 1073741824 )) GB"
  elif (( bytes >= 1048576 )); then
    echo "$(( bytes / 1048576 )) MB"
  elif (( bytes >= 1024 )); then
    echo "$(( bytes / 1024 )) KB"
  else
    echo "${bytes} B"
  fi
}

# ─── Validation ──────────────────────────────────────────────────────────────

# Validate that a port number is within the acceptable range (1024–65535).
# Exits with error if invalid.
# Usage: validate_port 8080
validate_port() {
  local port="${1:?validate_port: port number required}"

  # Must be a positive integer
  if ! [[ "${port}" =~ ^[0-9]+$ ]]; then
    err "Invalid port '${port}': must be a number between 1024 and 65535"
  fi

  if (( port < 1024 || port > 65535 )); then
    err "Invalid port '${port}': must be between 1024 and 65535"
  fi
}

# Validate that a version tag exists in a remote git repository.
# Uses `git ls-remote` to check without cloning.
# Usage: validate_version_tag <tag> <repo_url>
#   validate_version_tag "v1.2.3" "https://github.com/anonysec/koris.git"
validate_version_tag() {
  local tag="${1:?validate_version_tag: tag required}"
  local repo_url="${2:?validate_version_tag: repository URL required}"

  command -v git &>/dev/null || err "git is not installed"

  local refs
  refs=$(git ls-remote --tags --refs "${repo_url}" "refs/tags/${tag}" 2>/dev/null)

  if [[ -z "${refs}" ]]; then
    err "Tag '${tag}' not found in remote repository ${repo_url}"
  fi
}

# Management commands referenced info()/error(); map them onto the shared helpers.
info()  { log "$@"; }
error() { echo -e "${RED}[-]${NC} $*" >&2; }
red='\033[0;31m'; green='\033[0;32m'; yellow='\033[0;33m'; blue='\033[0;34m'; cyan='\033[0;36m'; bold='\033[1m'; dim='\033[2m'; plain='\033[0m'

INSTALL_DIR="/opt/koris"
PANEL_ENV="/opt/koris/panel.env"
NODE_ENV="/etc/knode/node.env"
COMPOSE_FILE="${INSTALL_DIR}/docker-compose.yml"

# Signal trapping for clean exit
trap 'echo ""; echo -e "${yellow}Operation cancelled.${plain}"; exit 130' INT TERM

is_panel() { [[ -f "$COMPOSE_FILE" ]] && command -v docker &>/dev/null; }
is_node()  { docker ps --format '{{.Names}}' 2>/dev/null | grep -qx knode; }
get_version() { cat "$INSTALL_DIR/VERSION" 2>/dev/null || echo "?"; }

panel_status() {
    docker inspect -f '{{.State.Status}}' koris 2>/dev/null || echo "not running"
}
node_status() {
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx knode; then
        echo "running"
    else
        echo "not running"
    fi
}

cmd_start() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }
    cd "$INSTALL_DIR" && docker compose up -d && info "Panel started"
}
cmd_stop() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }
    cd "$INSTALL_DIR" && docker compose down && info "Panel stopped"
}
cmd_restart() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }
    cd "$INSTALL_DIR" && docker compose restart && info "Panel restarted"
}

cmd_status() {
    local panel_port
    panel_port=$(grep -oP 'PANEL_PORT=\K.*' "$PANEL_ENV" 2>/dev/null || echo "2096")
    echo -e "${bold}${blue}KorisPanel${plain} v$(get_version)"
    echo "───────────────────────────────────"
    printf "  %-14s %s\n" "Panel:" "$(panel_status)"
    printf "  %-14s %s\n" "Node Agent:" "$(node_status)"
    if is_panel; then
        local addr=$(grep 'PANEL_ADDR' "$PANEL_ENV" 2>/dev/null | cut -d= -f2 | tr -d "'\"")
        local home=$(grep 'KORIS_HOME' "$PANEL_ENV" 2>/dev/null | cut -d= -f2 | tr -d "'\"")
        printf "  %-14s %s\n" "Listen:" "${addr:-?} (HTTPS)"
        printf "  %-14s %s\n" "Data dir:" "${home:-?}"
        if curl -fsS -k "https://localhost:${panel_port:-2096}/api/health" >/dev/null 2>&1 \
           || curl -fsS "http://127.0.0.1:${panel_port:-2096}/api/health" >/dev/null 2>&1; then
            printf "  %-14s ${green}%s${plain}\n" "Health:" "OK"
        else
            printf "  %-14s ${red}%s${plain}\n" "Health:" "FAIL"
        fi
    fi
    echo "───────────────────────────────────"
    printf "  %-14s %s\n" "CPU:" "$(nproc) cores"
    printf "  %-14s %s\n" "RAM:" "$(free -h | awk '/^Mem:/{print $3"/"$2}')"
    printf "  %-14s %s\n" "Disk:" "$(df -h / | awk 'NR==2{print $3"/"$2" ("$5")"}')"
}

cmd_logs() {
    cd "$INSTALL_DIR" && docker compose logs --tail 50
}

cmd_follow() {
    cd "$INSTALL_DIR" && exec docker compose logs -f
}

cmd_update() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }
    [[ ! -d "$INSTALL_DIR/.git" ]] && { error "Not a git install at $INSTALL_DIR"; exit 1; }

    # Parse --version flag
    local target_tag=""
    for arg in "$@"; do
        case "${arg}" in
            --version=*) target_tag="${arg#*=}" ;;
            *)           error "Unknown flag: ${arg}"; exit 1 ;;
        esac
    done

    cd "$INSTALL_DIR"

    # Read current version
    local old_version
    old_version=$(cat "$INSTALL_DIR/VERSION" 2>/dev/null || echo "unknown")

    # Store current version before modification (for rollback reference)
    mkdir -p /etc/koris
    echo "${old_version}" > /etc/koris/version

    # Fetch all tags and branches
    git fetch --all --tags --quiet 2>/dev/null

    if [[ -n "${target_tag}" ]]; then
        # Validate tag exists
        local tag_exists
        tag_exists=$(git tag -l "${target_tag}")
        if [[ -z "${tag_exists}" ]]; then
            # Try remote
            tag_exists=$(git ls-remote --tags origin "refs/tags/${target_tag}" 2>/dev/null)
            if [[ -z "${tag_exists}" ]]; then
                error "Tag '${target_tag}' not found in repository"; exit 1
            fi
        fi

        # Check if already at target version
        if [[ "${old_version}" == "${target_tag}" ]]; then
            info "Already at version ${target_tag}. Nothing to do."
            return
        fi

        git checkout "${target_tag}" --quiet 2>/dev/null || { error "Failed to checkout tag '${target_tag}'"; exit 1; }
    else
        # Pull latest from main
        git checkout main --quiet 2>/dev/null || true
        git pull origin main --quiet 2>/dev/null || { error "Failed to pull latest from main"; exit 1; }

        local new_version
        new_version=$(cat "$INSTALL_DIR/VERSION" 2>/dev/null || echo "unknown")

        # Check if already up to date
        if [[ "${old_version}" == "${new_version}" ]]; then
            info "Already up to date (v${new_version})."
            return
        fi
    fi

    local new_version
    new_version=$(cat "$INSTALL_DIR/VERSION" 2>/dev/null || echo "unknown")

    info "Updating v${old_version} → v${new_version}..."

    # Rebuild
    docker compose up -d --build || { error "Docker build/start failed"; exit 1; }

    # Display changelog (max 50 lines)
    if [[ "${old_version}" != "unknown" && "${new_version}" != "unknown" ]]; then
        local changelog
        changelog=$(git log --oneline "${old_version}..${new_version}" 2>/dev/null | head -50)
        if [[ -n "${changelog}" ]]; then
            echo ""
            echo -e "${cyan}Changelog (${old_version} → ${new_version}):${plain}"
            echo "${changelog}" | sed 's/^/  /'
            echo ""
        fi
    fi

    # Health check: poll every 2s for 60s
    info "Checking health..."
    local attempts=0
    local healthy=""
    local panel_port
    panel_port=$(grep -oP 'PANEL_PORT=\K.*' "$PANEL_ENV" 2>/dev/null || echo "2096")

    while [[ ${attempts} -lt 30 ]]; do
        if curl -fsS -k "https://localhost:${panel_port}/api/health" >/dev/null 2>&1 \
           || curl -fsS "http://127.0.0.1:${panel_port}/api/health" >/dev/null 2>&1; then
            healthy="yes"
            break
        fi
        sleep 2
        attempts=$((attempts + 1))
    done

    if [[ "${healthy}" == "yes" ]]; then
        info "Update complete: v${old_version} → v${new_version} ✓"
        echo "${new_version}" > /etc/koris/version
    else
        error "Health check failed after 60 seconds"
        echo ""
        echo -e "${red}Last 20 lines of container logs:${plain}"
        docker logs koris --tail 20 2>&1 | sed 's/^/  /'
        echo ""
        warn "Suggest: koris downgrade ${old_version}"
        exit 1
    fi

    # Update CLI self
    [[ -f "$INSTALL_DIR/panel/koris.sh" ]] && { cp "$INSTALL_DIR/panel/koris.sh" /usr/local/bin/koris 2>/dev/null; chmod +x /usr/local/bin/koris 2>/dev/null; }
}

cmd_uninstall() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }

    local keep_data=""
    for arg in "$@"; do
        case "${arg}" in
            --keep-data) keep_data="yes" ;;
            *)           error "Unknown flag: ${arg}"; exit 1 ;;
        esac
    done

    # Display summary
    echo -e "${red}${bold}KorisPanel Uninstall${plain}"
    echo ""
    echo "  The following will be removed:"
    echo "    • Docker containers: koris, koris-db, koris-pgadmin"
    echo "    • Docker images: koris project images"
    if [[ "${keep_data}" == "yes" ]]; then
        echo "    • Docker volumes: koris_panel-data, koris_pgadmin-data (DB preserved)"
    else
        echo "    • Docker volumes: koris_db-data, koris_panel-data, koris_pgadmin-data"
    fi
    echo "    • Installation directory: /opt/koris"
    echo "    • Configuration: /etc/koris"
    echo "    • CLI binary: /usr/local/bin/koris"
    echo "    • Certbot cron jobs for Koris"
    if docker ps -a --format '{{.Names}}' 2>/dev/null | grep -qx knode; then
        echo "    • knode container, image, and /etc/knode"
    fi
    echo ""

    read -rp "Type 'yes' to confirm uninstall: " confirm
    if [[ "${confirm}" != "yes" ]]; then
        info "Uninstall cancelled."
        return
    fi

    local -a failures=()

    # 1. Stop and remove Docker Compose stack
    info "Stopping containers..."
    if [[ -d "$INSTALL_DIR" ]]; then
        cd "$INSTALL_DIR"
        docker compose down --remove-orphans 2>/dev/null || failures+=("docker compose down")
    fi
    # Force-remove individual containers if still present
    for ctr in koris koris-db koris-pgadmin; do
        if docker ps -a --format '{{.Names}}' 2>/dev/null | grep -qx "${ctr}"; then
            docker rm -f "${ctr}" 2>/dev/null || failures+=("remove container ${ctr}")
        fi
    done

    # 2. Remove project images
    info "Removing project images..."
    local images
    images=$(docker images --format '{{.ID}} {{.Repository}}' 2>/dev/null | awk '$2 ~ /^koris/ {print $1}')
    images+=" $(docker images --filter "label=com.docker.compose.project=koris" -q 2>/dev/null)"
    for img in $(echo "${images}" | tr ' ' '\n' | sort -u | grep -v '^$'); do
        docker rmi -f "${img}" 2>/dev/null || failures+=("remove image ${img}")
    done

    # 3. Remove volumes
    info "Removing volumes..."
    if [[ "${keep_data}" == "yes" ]]; then
        for vol in koris_panel-data koris_pgadmin-data; do
            docker volume rm "${vol}" 2>/dev/null || failures+=("remove volume ${vol}")
        done
    else
        for vol in koris_db-data koris_panel-data koris_pgadmin-data; do
            docker volume rm "${vol}" 2>/dev/null || failures+=("remove volume ${vol}")
        done
    fi

    # 4. Remove installation directory
    info "Removing /opt/koris..."
    rm -rf /opt/koris 2>/dev/null || failures+=("remove /opt/koris")

    # 5. Remove configuration
    if [[ "${keep_data}" != "yes" ]]; then
        info "Removing /etc/koris..."
        rm -rf /etc/koris 2>/dev/null || failures+=("remove /etc/koris")
    fi

    # 6. Remove CLI binary
    rm -f /usr/local/bin/koris 2>/dev/null || failures+=("remove /usr/local/bin/koris")

    # 7. Remove certbot cron jobs
    if crontab -l 2>/dev/null | grep -q "koris\|KorisPanel"; then
        crontab -l 2>/dev/null | grep -v "koris\|KorisPanel" | crontab - 2>/dev/null || failures+=("remove certbot cron")
    fi

    # 8. Handle knode if present
    if docker ps -a --format '{{.Names}}' 2>/dev/null | grep -qx knode; then
        info "Removing knode..."
        docker stop knode 2>/dev/null || true
        docker rm -f knode 2>/dev/null || failures+=("remove knode container")
        docker rmi knode:latest 2>/dev/null || failures+=("remove knode image")
        rm -rf /etc/knode 2>/dev/null || failures+=("remove /etc/knode")
    fi

    # Summary
    echo ""
    if [[ ${#failures[@]} -gt 0 ]]; then
        warn "Uninstall completed with ${#failures[@]} error(s):"
        for f in "${failures[@]}"; do
            echo -e "    ${red}✗${plain} ${f}"
        done
    else
        info "KorisPanel completely uninstalled."
    fi
    if [[ "${keep_data}" == "yes" ]]; then
        info "Database volume and backups preserved."
    fi
}

cmd_config() {
    is_panel && { echo -e "${cyan}Panel Config:${plain}"; grep -v 'SECRET\|PASSWORD\|TOKEN' "$PANEL_ENV" 2>/dev/null | sed 's/^/  /'; echo "  (secrets hidden)"; }
    is_node  && { echo -e "${cyan}Node Config:${plain}"; grep -v 'TOKEN' "$NODE_ENV" 2>/dev/null | sed 's/^/  /'; echo "  (token hidden)"; }
}


cmd_clean() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }
    docker info &>/dev/null || { error "Docker daemon not available"; exit 1; }

    local do_volumes="" do_include_db="" do_all="" do_force=""
    for arg in "$@"; do
        case "${arg}" in
            --volumes)    do_volumes="yes" ;;
            --include-db) do_include_db="yes" ;;
            --all)        do_all="yes" ;;
            --force)      do_force="yes" ;;
            *)            error "Unknown flag: ${arg}"; exit 1 ;;
        esac
    done

    # --all implies everything
    if [[ "${do_all}" == "yes" ]]; then
        if [[ "${do_force}" != "yes" ]]; then
            echo -e "${red}This will remove ALL project volumes (including database), images, and build cache.${plain}"
            read -rp "Type 'yes' to confirm: " confirm
            [[ "${confirm}" != "yes" ]] && { info "Cancelled."; return; }
        fi
        do_volumes="yes"
        do_include_db="yes"
    fi

    local total_reclaimed=0

    # Remove dangling/project images
    info "Removing project images..."
    local img_output
    img_output=$(docker images --filter "label=com.docker.compose.project=koris" -q 2>/dev/null)
    # Also get images starting with "koris"
    local koris_images
    koris_images=$(docker images --format '{{.ID}} {{.Repository}}' | awk '$2 ~ /^koris/ {print $1}')
    local all_images
    all_images=$(echo -e "${img_output}\n${koris_images}" | sort -u | grep -v '^$')

    if [[ -n "${all_images}" ]]; then
        for img_id in ${all_images}; do
            local size
            size=$(docker image inspect "${img_id}" --format='{{.Size}}' 2>/dev/null || echo "0")
            if docker rmi -f "${img_id}" &>/dev/null; then
                total_reclaimed=$((total_reclaimed + size))
            fi
        done
    fi

    # Prune build cache
    info "Pruning Docker build cache..."
    local cache_output
    cache_output=$(docker builder prune -f 2>&1 || true)
    # Extract reclaimed bytes from builder prune output if possible
    local cache_bytes
    cache_bytes=$(echo "${cache_output}" | grep -oP 'Total:\s+\K[0-9]+' || echo "0")
    if [[ -n "${cache_bytes}" && "${cache_bytes}" =~ ^[0-9]+$ ]]; then
        total_reclaimed=$((total_reclaimed + cache_bytes))
    fi

    # Remove volumes if requested
    if [[ "${do_volumes}" == "yes" ]]; then
        info "Removing project volumes..."
        for vol in koris_panel-data koris_pgadmin-data; do
            # Check if volume is in use
            local container_using
            container_using=$(docker ps --filter "volume=${vol}" --format '{{.Names}}' 2>/dev/null | head -1)
            if [[ -n "${container_using}" ]]; then
                warn "Volume '${vol}' is in use by container '${container_using}' — skipping"
                continue
            fi
            if docker volume rm "${vol}" &>/dev/null; then
                info "Removed volume: ${vol}"
            fi
        done

        if [[ "${do_include_db}" == "yes" ]]; then
            local container_using
            container_using=$(docker ps --filter "volume=koris_db-data" --format '{{.Names}}' 2>/dev/null | head -1)
            if [[ -n "${container_using}" ]]; then
                warn "Volume 'koris_db-data' is in use by container '${container_using}' — skipping"
            elif docker volume rm koris_db-data &>/dev/null; then
                info "Removed volume: koris_db-data"
            fi
        fi
    fi

    info "Clean complete. Space reclaimed: $(human_bytes ${total_reclaimed})"
}

cmd_db() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }

    # Check koris-db container is running
    local db_state
    db_state=$(docker inspect -f '{{.State.Status}}' koris-db 2>/dev/null || echo "not found")
    [[ "${db_state}" != "running" ]] && { error "Database container 'koris-db' is not running (state: ${db_state})"; exit 1; }

    local subcmd="${1:-}"
    shift 2>/dev/null || true

    case "${subcmd}" in
        backup)  cmd_db_backup "$@" ;;
        restore) cmd_db_restore "$@" ;;
        migrate) cmd_db_migrate ;;
        reset)   cmd_db_reset ;;
        shell)   cmd_db_shell ;;
        status)  cmd_db_status ;;
        *)       error "Usage: koris db [backup|restore|migrate|reset|shell|status]"; exit 1 ;;
    esac
}

cmd_db_backup() {
    local backup_dir="/var/backups/koris"

    # Parse --path flag
    for arg in "$@"; do
        case "${arg}" in
            --path=*) backup_dir="${arg#*=}" ;;
            *)        error "Unknown flag: ${arg}"; exit 1 ;;
        esac
    done

    # Validate backup directory
    if [[ ! -d "${backup_dir}" ]]; then
        mkdir -p "${backup_dir}" 2>/dev/null || { error "Cannot create directory: ${backup_dir}"; exit 1; }
    fi
    [[ -w "${backup_dir}" ]] || { error "Directory not writable: ${backup_dir}"; exit 1; }

    local db_name db_user
    db_name=$(grep -oP 'POSTGRES_DB=\K.*' "$PANEL_ENV" 2>/dev/null || echo "koris")
    db_user=$(grep -oP 'POSTGRES_USER=\K.*' "$PANEL_ENV" 2>/dev/null || echo "koris")

    local timestamp
    timestamp=$(date -u +"%Y%m%d-%H%M%S")
    local backup_file="${backup_dir}/koris-${timestamp}.sql.gz"

    info "Creating database backup..."
    docker exec koris-db pg_dump -U "${db_user}" "${db_name}" | gzip > "${backup_file}" \
        || { rm -f "${backup_file}"; error "Backup failed"; exit 1; }

    local size
    size=$(stat -c%s "${backup_file}" 2>/dev/null || echo "0")
    info "Backup saved: ${backup_file} ($(human_bytes ${size}))"
}

cmd_db_restore() {
    local restore_file="${1:-}"
    local db_name db_user
    db_name=$(grep -oP 'POSTGRES_DB=\K.*' "$PANEL_ENV" 2>/dev/null || echo "koris")
    db_user=$(grep -oP 'POSTGRES_USER=\K.*' "$PANEL_ENV" 2>/dev/null || echo "koris")

    # If no file specified, list available backups
    if [[ -z "${restore_file}" ]]; then
        local backup_dir="/var/backups/koris"
        if [[ ! -d "${backup_dir}" ]] || [[ -z "$(ls -A "${backup_dir}" 2>/dev/null)" ]]; then
            error "No backups found in ${backup_dir}"; exit 1
        fi
        info "Available backups:"
        local i=1
        local -a files=()
        while IFS= read -r f; do
            files+=("${f}")
            printf "  ${cyan}%d)${plain} %s (%s)\n" "$i" "$(basename "${f}")" "$(human_bytes $(stat -c%s "${f}" 2>/dev/null || echo 0))"
            i=$((i + 1))
        done < <(ls -t "${backup_dir}"/koris-*.sql.gz 2>/dev/null)

        [[ ${#files[@]} -eq 0 ]] && { error "No .sql.gz backups found"; exit 1; }

        echo ""
        read -rp "$(echo -e "${cyan}Select backup number: ${plain}")" selection
        if ! [[ "${selection}" =~ ^[0-9]+$ ]] || (( selection < 1 || selection > ${#files[@]} )); then
            error "Invalid selection"; exit 1
        fi
        restore_file="${files[$((selection - 1))]}"
    fi

    # Validate file exists
    [[ -f "${restore_file}" ]] || { error "File not found: ${restore_file}"; exit 1; }
    # Validate file is valid gzip
    gzip -t "${restore_file}" 2>/dev/null || { error "File is not a valid gzip archive: ${restore_file}"; exit 1; }

    # Confirmation prompt
    echo -e "${red}This will OVERWRITE the current database with the backup.${plain}"
    read -rp "Type 'yes' to confirm: " confirm
    [[ "${confirm}" != "yes" ]] && { info "Cancelled."; return; }

    info "Restoring database from: $(basename "${restore_file}")..."

    # Drop and recreate database
    docker exec koris-db psql -U "${db_user}" -d postgres -c "DROP DATABASE IF EXISTS ${db_name};" 2>/dev/null
    docker exec koris-db psql -U "${db_user}" -d postgres -c "CREATE DATABASE ${db_name} OWNER ${db_user};" 2>/dev/null

    # Restore dump
    gunzip -c "${restore_file}" | docker exec -i koris-db psql -U "${db_user}" -d "${db_name}" >/dev/null 2>&1 \
        || { error "Restore failed"; exit 1; }

    info "Database restored successfully from $(basename "${restore_file}")"
}

cmd_db_migrate() {
    local db_name db_user
    db_name=$(grep -oP 'POSTGRES_DB=\K.*' "$PANEL_ENV" 2>/dev/null || echo "koris")
    db_user=$(grep -oP 'POSTGRES_USER=\K.*' "$PANEL_ENV" 2>/dev/null || echo "koris")

    info "Running database migrations..."
    local output
    output=$(docker exec koris /app/migrate-db 2>&1) || { error "Migration failed: ${output}"; exit 1; }

    # Try to extract migration count from output
    local count
    count=$(echo "${output}" | grep -oP '\d+ migration' | grep -oP '\d+' || echo "")
    if [[ -n "${count}" ]]; then
        info "Migrations complete: ${count} applied"
    else
        info "Migrations complete"
        echo "${output}" | tail -5
    fi
}

cmd_db_reset() {
    local db_name db_user
    db_name=$(grep -oP 'POSTGRES_DB=\K.*' "$PANEL_ENV" 2>/dev/null || echo "koris")
    db_user=$(grep -oP 'POSTGRES_USER=\K.*' "$PANEL_ENV" 2>/dev/null || echo "koris")

    echo -e "${red}WARNING: This will DROP and recreate the database, then run all migrations from scratch.${plain}"
    echo -e "${red}ALL DATA WILL BE LOST.${plain}"
    read -rp "Type 'yes' to confirm: " confirm
    [[ "${confirm}" != "yes" ]] && { info "Cancelled."; return; }

    info "Dropping database '${db_name}'..."
    docker exec koris-db psql -U "${db_user}" -d postgres -c "DROP DATABASE IF EXISTS ${db_name};" 2>/dev/null
    docker exec koris-db psql -U "${db_user}" -d postgres -c "CREATE DATABASE ${db_name} OWNER ${db_user};" 2>/dev/null
    info "Database recreated. Running migrations..."

    # Run migrations
    local output
    output=$(docker exec koris /app/migrate-db 2>&1) || { error "Migration failed: ${output}"; exit 1; }
    info "Database reset complete — all migrations applied from scratch"
}

cmd_db_shell() {
    local db_name db_user
    db_name=$(grep -oP 'POSTGRES_DB=\K.*' "$PANEL_ENV" 2>/dev/null || echo "koris")
    db_user=$(grep -oP 'POSTGRES_USER=\K.*' "$PANEL_ENV" 2>/dev/null || echo "koris")

    info "Opening psql shell (Ctrl+D or \\q to exit)..."
    exec docker exec -it koris-db psql -U "${db_user}" -d "${db_name}"
}

cmd_db_status() {
    local db_name db_user
    db_name=$(grep -oP 'POSTGRES_DB=\K.*' "$PANEL_ENV" 2>/dev/null || echo "koris")
    db_user=$(grep -oP 'POSTGRES_USER=\K.*' "$PANEL_ENV" 2>/dev/null || echo "koris")

    echo -e "${bold}${cyan}Database Status${plain}"
    echo "───────────────────────────────────"

    # Database size
    local db_size
    db_size=$(docker exec koris-db psql -U "${db_user}" -d "${db_name}" -t -c "SELECT pg_size_pretty(pg_database_size('${db_name}'));" 2>/dev/null | xargs)
    printf "  %-20s %s\n" "Size:" "${db_size:-unknown}"

    # Active connections
    local connections
    connections=$(docker exec koris-db psql -U "${db_user}" -d "${db_name}" -t -c "SELECT count(*) FROM pg_stat_activity WHERE datname='${db_name}';" 2>/dev/null | xargs)
    printf "  %-20s %s\n" "Connections:" "${connections:-unknown}"

    # TimescaleDB version
    local tsdb_version
    tsdb_version=$(docker exec koris-db psql -U "${db_user}" -d "${db_name}" -t -c "SELECT extversion FROM pg_extension WHERE extname='timescaledb';" 2>/dev/null | xargs)
    printf "  %-20s %s\n" "TimescaleDB:" "${tsdb_version:-not installed}"

    # Replication status
    local replication
    replication=$(docker exec koris-db psql -U "${db_user}" -d "${db_name}" -t -c "SELECT count(*) FROM pg_stat_replication;" 2>/dev/null | xargs)
    if [[ "${replication}" == "0" || -z "${replication}" ]]; then
        printf "  %-20s %s\n" "Replication:" "none (standalone)"
    else
        printf "  %-20s %s replicas\n" "Replication:" "${replication}"
    fi

    echo "───────────────────────────────────"
}

cmd_pgadmin() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }
    docker info &>/dev/null || { error "Docker daemon not available"; exit 1; }

    local subcmd="${1:-}"
    shift 2>/dev/null || true

    case "${subcmd}" in
        status)         pgadmin_status ;;
        enable)         pgadmin_enable ;;
        disable)        pgadmin_disable ;;
        url)            pgadmin_url ;;
        reset-password) pgadmin_reset_password ;;
        port)           pgadmin_port "$@" ;;
        *)              error "Usage: koris pgadmin [status|enable|disable|url|reset-password|port <number>]"; exit 1 ;;
    esac
}

pgadmin_status() {
    local state
    state=$(docker inspect -f '{{.State.Status}}' koris-pgadmin 2>/dev/null || echo "not found")
    if [[ "${state}" == "running" ]]; then
        local port
        port=$(grep -oP 'PGADMIN_PORT=\K.*' "$PANEL_ENV" 2>/dev/null || echo "5050")
        local ip
        ip=$(curl -fsS4 --max-time 3 https://api.ipify.org 2>/dev/null || hostname -I | awk '{print $1}')
        echo -e "  ${green}●${plain} pgAdmin is ${green}running${plain}"
        echo -e "  URL:  ${cyan}http://${ip}:${port}${plain}"
        echo -e "  Port: ${port}"
    else
        echo -e "  ${red}●${plain} pgAdmin is ${red}${state}${plain}"
    fi
}

pgadmin_enable() {
    local state
    state=$(docker inspect -f '{{.State.Status}}' koris-pgadmin 2>/dev/null || echo "not found")
    if [[ "${state}" == "running" ]]; then
        info "pgAdmin is already running."
        return
    fi

    info "Starting pgAdmin..."
    docker start koris-pgadmin 2>/dev/null || { cd "$INSTALL_DIR" && docker compose up -d koris-pgadmin; }
    docker update --restart unless-stopped koris-pgadmin 2>/dev/null || true

    # Wait up to 30s
    local attempts=0
    while [[ ${attempts} -lt 15 ]]; do
        state=$(docker inspect -f '{{.State.Status}}' koris-pgadmin 2>/dev/null || echo "")
        [[ "${state}" == "running" ]] && break
        sleep 2
        attempts=$((attempts + 1))
    done

    if [[ "${state}" == "running" ]]; then
        local port ip
        port=$(grep -oP 'PGADMIN_PORT=\K.*' "$PANEL_ENV" 2>/dev/null || echo "5050")
        ip=$(curl -fsS4 --max-time 3 https://api.ipify.org 2>/dev/null || hostname -I | awk '{print $1}')
        info "pgAdmin is running at http://${ip}:${port}"
    else
        error "pgAdmin failed to start within 30 seconds"
    fi
}

pgadmin_disable() {
    docker stop koris-pgadmin 2>/dev/null || true
    docker update --restart no koris-pgadmin 2>/dev/null || true
    info "pgAdmin stopped and autostart disabled."
}

pgadmin_url() {
    local state
    state=$(docker inspect -f '{{.State.Status}}' koris-pgadmin 2>/dev/null || echo "not found")
    if [[ "${state}" != "running" ]]; then
        error "pgAdmin is not running. Start it with: koris pgadmin enable"; exit 1
    fi
    local port ip
    port=$(grep -oP 'PGADMIN_PORT=\K.*' "$PANEL_ENV" 2>/dev/null || echo "5050")
    ip=$(curl -fsS4 --max-time 3 https://api.ipify.org 2>/dev/null || hostname -I | awk '{print $1}')
    echo "http://${ip}:${port}"
}

pgadmin_reset_password() {
    local state
    state=$(docker inspect -f '{{.State.Status}}' koris-pgadmin 2>/dev/null || echo "not found")
    [[ "${state}" != "running" ]] && { error "pgAdmin is not running"; exit 1; }

    read -rsp "$(echo -e "${cyan}New pgAdmin password (min 8 chars): ${plain}")" new_pass
    echo ""
    if [[ ${#new_pass} -lt 8 ]]; then
        error "Password must be at least 8 characters"; exit 1
    fi

    # Update panel.env
    sed -i "s|^PGADMIN_PASSWORD=.*|PGADMIN_PASSWORD=${new_pass}|" "$PANEL_ENV"

    # Restart pgAdmin with new password
    docker stop koris-pgadmin 2>/dev/null
    cd "$INSTALL_DIR" && docker compose up -d koris-pgadmin
    info "pgAdmin password updated and service restarted."
}

pgadmin_port() {
    local new_port="${1:-}"
    [[ -z "${new_port}" ]] && { error "Usage: koris pgadmin port <number>"; exit 1; }

    # Validate port range
    if ! [[ "${new_port}" =~ ^[0-9]+$ ]] || (( new_port < 1024 || new_port > 65535 )); then
        error "Invalid port '${new_port}': must be between 1024 and 65535"; exit 1
    fi

    # Update panel.env
    sed -i "s|^PGADMIN_PORT=.*|PGADMIN_PORT=${new_port}|" "$PANEL_ENV"

    # Restart pgAdmin with new port
    info "Updating pgAdmin port to ${new_port}..."
    cd "$INSTALL_DIR"
    docker compose stop koris-pgadmin 2>/dev/null
    docker compose up -d koris-pgadmin

    # Wait up to 30s
    local attempts=0
    while [[ ${attempts} -lt 15 ]]; do
        local state
        state=$(docker inspect -f '{{.State.Status}}' koris-pgadmin 2>/dev/null || echo "")
        [[ "${state}" == "running" ]] && break
        sleep 2
        attempts=$((attempts + 1))
    done

    local ip
    ip=$(curl -fsS4 --max-time 3 https://api.ipify.org 2>/dev/null || hostname -I | awk '{print $1}')
    info "pgAdmin is now available at http://${ip}:${new_port}"
}

cmd_reinstall() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }

    local do_clean=""
    for arg in "$@"; do
        case "${arg}" in
            --clean) do_clean="yes" ;;
            *)       error "Unknown flag: ${arg}"; exit 1 ;;
        esac
    done

    # Verify panel.env exists with POSTGRES_PASSWORD
    if [[ ! -f "$PANEL_ENV" ]]; then
        error "Configuration file not found: $PANEL_ENV"
        echo "  Cannot reinstall without existing configuration."
        exit 1
    fi
    local db_pass
    db_pass=$(grep -oP 'POSTGRES_PASSWORD=\K.*' "$PANEL_ENV" 2>/dev/null || true)
    if [[ -z "${db_pass}" ]]; then
        error "POSTGRES_PASSWORD not found in $PANEL_ENV"
        echo "  Cannot reinstall without database password."
        exit 1
    fi

    info "Reinstalling KorisPanel (database data preserved)..."

    # 1. Stop and remove containers
    info "Stopping and removing containers..."
    cd "$INSTALL_DIR" 2>/dev/null || true
    docker compose down --remove-orphans 2>/dev/null || true
    for ctr in koris koris-db koris-pgadmin; do
        docker rm -f "${ctr}" 2>/dev/null || true
    done

    # 2. Remove project images
    info "Removing project images..."
    local images
    images=$(docker images --format '{{.ID}} {{.Repository}}' 2>/dev/null | awk '$2 ~ /^koris/ {print $1}')
    images+=" $(docker images --filter "label=com.docker.compose.project=koris" -q 2>/dev/null)"
    for img in $(echo "${images}" | tr ' ' '\n' | sort -u | grep -v '^$'); do
        docker rmi -f "${img}" 2>/dev/null || true
    done

    # 3. Remove panel-data and pgadmin-data volumes (preserve db-data)
    docker volume rm koris_panel-data koris_pgadmin-data 2>/dev/null || true

    # 4. Prune build cache if --clean
    if [[ "${do_clean}" == "yes" ]]; then
        info "Pruning Docker build cache..."
        docker builder prune -f 2>/dev/null || true
    fi

    # 5. Pull latest source
    info "Pulling latest source from main..."
    if [[ -d "$INSTALL_DIR/.git" ]]; then
        git -C "$INSTALL_DIR" fetch origin main --quiet 2>/dev/null || { error "Git fetch failed"; exit 1; }
        git -C "$INSTALL_DIR" checkout main --quiet 2>/dev/null || true
        git -C "$INSTALL_DIR" reset --hard origin/main --quiet 2>/dev/null || { error "Git pull failed"; exit 1; }
    else
        error "Source directory $INSTALL_DIR is not a git repository"
        exit 1
    fi

    # 6. Rebuild all containers
    info "Building containers..."
    cd "$INSTALL_DIR"
    docker compose build || { error "Docker build failed — database data is intact"; exit 1; }
    docker compose up -d || { error "Docker Compose failed to start services"; exit 1; }

    # 7. Health check: poll every 5s for 60s
    info "Checking health..."
    local attempts=0
    local healthy=""
    local panel_port
    panel_port=$(grep -oP 'PANEL_PORT=\K.*' "$PANEL_ENV" 2>/dev/null || echo "2096")

    while [[ ${attempts} -lt 12 ]]; do
        if curl -fsS -k "https://localhost:${panel_port}/api/health" >/dev/null 2>&1 \
           || curl -fsS "http://127.0.0.1:${panel_port}/api/health" >/dev/null 2>&1; then
            healthy="yes"
            break
        fi
        sleep 5
        attempts=$((attempts + 1))
    done

    if [[ "${healthy}" == "yes" ]]; then
        info "Reinstall complete — panel is healthy ✓"
    else
        error "Health check timed out after 60 seconds"
        docker logs koris --tail 20 2>&1 | sed 's/^/  /'
        exit 1
    fi

    # Update CLI self
    [[ -f "$INSTALL_DIR/panel/koris.sh" ]] && { cp "$INSTALL_DIR/panel/koris.sh" /usr/local/bin/koris 2>/dev/null; chmod +x /usr/local/bin/koris 2>/dev/null; }
}

cmd_downgrade() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }

    local target_tag="${1:-}"
    if [[ -z "${target_tag}" ]]; then
        error "Usage: koris downgrade <version-tag>"
        echo "  Example: koris downgrade v1.2.0"
        exit 1
    fi

    # Don't accept flags as the tag
    if [[ "${target_tag}" == --* ]]; then
        error "Usage: koris downgrade <version-tag>"
        echo "  Example: koris downgrade v1.2.0"
        exit 1
    fi

    info "Downgrading to version: ${target_tag}"
    info "This will rebuild the panel at the specified version while preserving database data."
    echo ""

    # Invoke the in-file installer with --version and --reinstall
    do_install --version="${target_tag}" --reinstall
}

show_menu() {
    clear
    echo -e "${bold}${blue}KorisPanel${plain} v$(get_version)    Panel: $(panel_status)  Node: $(node_status)"
    echo ""
    echo -e "  ${green}1.${plain}  Start               ${green}10.${plain} Disable autostart"
    echo -e "  ${green}2.${plain}  Stop                ${green}11.${plain} Uninstall"
    echo -e "  ${green}3.${plain}  Restart             ${green}12.${plain} SSL Certificate"
    echo -e "  ${green}4.${plain}  Status              ${green}13.${plain} Clean"
    echo -e "  ${green}5.${plain}  Logs                ${green}14.${plain} DB Management"
    echo -e "  ${green}6.${plain}  Live logs           ${green}15.${plain} pgAdmin Management"
    echo -e "  ${green}7.${plain}  Update              ${green}16.${plain} Reinstall"
    echo -e "  ${green}8.${plain}  Config              ${green}17.${plain} Downgrade"
    echo -e "  ${green}9.${plain}  Enable autostart    ${green}0.${plain}  Exit"
    echo ""
    read -rp "$(echo -e "${cyan}Choose [0-17]: ${plain}")" ch
    case "$ch" in
        1)  cmd_start ;;
        2)  cmd_stop ;;
        3)  cmd_restart ;;
        4)  cmd_status ;;
        5)  cmd_logs ;;
        6)  cmd_follow ;;
        7)  cmd_update ;;
        8)  cmd_config ;;
        9)  docker update --restart unless-stopped koris koris-db koris-pgadmin 2>/dev/null; info "Autostart enabled." ;;
        10) docker update --restart no koris koris-db koris-pgadmin 2>/dev/null; info "Autostart disabled." ;;
        11) cmd_uninstall ;;
        13) menu_clean ;;
        14) menu_db ;;
        15) menu_pgadmin ;;
        16) menu_reinstall ;;
        17) menu_downgrade ;;
        0)  exit 0 ;;
        *)  warn "Invalid selection. Enter a number 0-17." ;;
    esac
    echo ""; read -rp "Press Enter to continue..." _; show_menu
}

menu_db() {
    echo ""
    echo -e "${bold}${cyan}Database Management${plain}"
    echo ""
    echo -e "  ${green}1.${plain} Backup"
    echo -e "  ${green}2.${plain} Restore"
    echo -e "  ${green}3.${plain} Migrate"
    echo -e "  ${green}4.${plain} Reset"
    echo -e "  ${green}5.${plain} Shell"
    echo -e "  ${green}6.${plain} Status"
    echo -e "  ${green}0.${plain} Back"
    echo ""
    read -rp "$(echo -e "${cyan}Choose [0-6]: ${plain}")" db_ch
    case "$db_ch" in
        1) cmd_db backup ;;
        2) cmd_db restore ;;
        3) cmd_db migrate ;;
        4) cmd_db reset ;;
        5) cmd_db shell ;;
        6) cmd_db status ;;
        0) return ;;
        *) warn "Invalid selection. Enter a number 0-6." ;;
    esac
}

menu_pgadmin() {
    echo ""
    echo -e "${bold}${cyan}pgAdmin Management${plain}"
    echo ""
    echo -e "  ${green}1.${plain} Status"
    echo -e "  ${green}2.${plain} Enable"
    echo -e "  ${green}3.${plain} Disable"
    echo -e "  ${green}4.${plain} URL"
    echo -e "  ${green}5.${plain} Reset password"
    echo -e "  ${green}6.${plain} Change port"
    echo -e "  ${green}0.${plain} Back"
    echo ""
    read -rp "$(echo -e "${cyan}Choose [0-6]: ${plain}")" pg_ch
    case "$pg_ch" in
        1) cmd_pgadmin status ;;
        2) cmd_pgadmin enable ;;
        3) cmd_pgadmin disable ;;
        4) cmd_pgadmin url ;;
        5) cmd_pgadmin reset-password ;;
        6)
            read -rp "$(echo -e "${cyan}New port number: ${plain}")" new_port
            cmd_pgadmin port "${new_port}"
            ;;
        0) return ;;
        *) warn "Invalid selection. Enter a number 0-6." ;;
    esac
}

menu_clean() {
    echo ""
    echo -e "${bold}${cyan}Clean Docker Artifacts${plain}"
    echo ""
    echo -e "  ${green}1.${plain} Basic clean (remove images and build cache)"
    echo -e "  ${green}2.${plain} Clean with volumes (remove images, cache, panel+pgadmin volumes)"
    echo -e "  ${green}3.${plain} Full clean (remove everything including database volume)"
    echo -e "  ${green}0.${plain} Cancel"
    echo ""
    read -rp "$(echo -e "${cyan}Choose [0-3]: ${plain}")" clean_ch
    case "$clean_ch" in
        1) cmd_clean ;;
        2) cmd_clean --volumes ;;
        3) cmd_clean --all ;;
        0) return ;;
        *) warn "Invalid selection. Enter a number 0-3." ;;
    esac
}

menu_reinstall() {
    echo ""
    echo -e "${yellow}This will stop all containers, remove images, and rebuild from source.${plain}"
    echo -e "${yellow}Database data will be preserved.${plain}"
    echo ""
    read -rp "$(echo -e "${cyan}Proceed with reinstall? [y/N]: ${plain}")" confirm
    if [[ "${confirm}" =~ ^[yY] ]]; then
        cmd_reinstall
    else
        info "Cancelled."
    fi
}

menu_downgrade() {
    echo ""
    read -rp "$(echo -e "${cyan}Target version tag (e.g. v1.2.0): ${plain}")" target_ver
    if [[ -z "${target_ver}" ]]; then
        warn "No version specified."
        return
    fi
    echo ""
    echo -e "${yellow}This will rebuild the panel at version ${target_ver}.${plain}"
    read -rp "$(echo -e "${cyan}Confirm downgrade? [y/N]: ${plain}")" confirm
    if [[ "${confirm}" =~ ^[yY] ]]; then
        cmd_downgrade "${target_ver}"
    else
        info "Cancelled."
    fi
}

REPO="anonysec/koris"
KNODE_REPO="anonysec/knode"
INSTALL_DIR="/opt/koris"
CONFIG_DIR="/opt/koris"     # overridden at install time to match KORIS_HOME

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'


banner() {
  echo -e "${BOLD}${CYAN}"
  cat << 'EOF'
  ██╗  ██╗ ██████╗ ██████╗ ██╗███████╗
  ██║ ██╔╝██╔═══██╗██╔══██╗██║██╔════╝
  █████╔╝ ██║   ██║██████╔╝██║███████╗
  ██╔═██╗ ██║   ██║██╔══██╗██║╚════██║
  ██║  ██╗╚██████╔╝██║  ██║██║███████║
  ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝╚══════╝
EOF
  echo -e "${NC} ${GREEN}KorisPanel — VPN Management Panel Installer${NC}\n"
}

detect_os() {
  [[ -f /etc/os-release ]] || err "Unsupported OS: /etc/os-release not found"
  local os_id os_version
  os_id=$(. /etc/os-release && echo "$ID")
  os_version=$(. /etc/os-release && echo "$VERSION_ID")
  case "${os_id}" in
    ubuntu|debian) log "Detected ${os_id} ${os_version}" ;;
    *) err "Unsupported: ${os_id} ${os_version}. Need Ubuntu 22.04+ or Debian 12+" ;;
  esac
}

gen_secret() { openssl rand -hex "${1:-32}" 2>/dev/null || head -c "${1:-32}" /dev/urandom | od -An -tx1 | tr -d ' \n'; }

# --- Parse flags ---
EDITION="full"
PANEL_PORT="2096"          # single panel port (HTTPS); blank in prompts => 2096
KORIS_HOME="/opt/koris"     # unified dir: env, certs, db, acme (root + user + docker)
DOMAIN=""
DB_NAME="koris"
DB_USER="koris"
DB_PASS=""
WITH_KNODE="yes"
TLS_MODE="selfsigned"       # dev default; installer prompts for a real cert
CERT_PATH="${KORIS_HOME}/certs/cert.pem"
KEY_PATH="${KORIS_HOME}/certs/key.pem"
IMAGE_TAG=""
# URL scheme — configurable at setup, editable later via panel.env
ADMIN_PATH="/admin/"       # served at https://<host>/admin/
PORTAL_PATH="/account/"    # served at https://<host>/account/
ADMIN_HOST=""              # optional: subdomain override, e.g. admin.example.com
PORTAL_HOST=""             # optional: subdomain override, e.g. account.example.com

# Installation mode: "release" (pull pre-built image, fast) or "source" (git clone + build)
INSTALL_MODE="release"
# Container registry to pull from when INSTALL_MODE=release.
IMAGE_REGISTRY="ghcr.io/anonysec/koris"
FORCE_REINSTALL=""
parse_args() {
  for arg in "$@"; do
    case "${arg}" in
      --native)       err "Native mode is no longer supported. Only Docker deployment is available. Remove the --native flag and re-run." ;;
      --lite)         EDITION="lite" ;;
      --full)         EDITION="full" ;;
      --port=*)       PANEL_PORT="${arg#*=}" ;;
      --home=*)       KORIS_HOME="${arg#*=}" ;;
      --domain=*)     DOMAIN="${arg#*=}" ;;
      --ssl=*)        TLS_MODE="${arg#*=}" ;;      # domain | ip | custom | selfsigned
      --ssl-target=*) SSL_TARGET="${arg#*=}" ;;    # domain or IP for auto SSL
      --cert-path=*)  CERT_PATH="${arg#*=}" ;;
      --key-path=*)   KEY_PATH="${arg#*=}" ;;
      --no-knode)     WITH_KNODE="no" ;;
      --uninstall)    uninstall; exit 0 ;;
      --version=*)    IMAGE_TAG="${arg#*=}" ;;
      --reinstall)    FORCE_REINSTALL="yes" ;;
      --admin-path=*)   ADMIN_PATH="${arg#*=}" ;;
      --portal-path=*)  PORTAL_PATH="${arg#*=}" ;;
      --admin-host=*)   ADMIN_HOST="${arg#*=}" ;;
      --portal-host=*)  PORTAL_HOST="${arg#*=}" ;;
      --from-source)    INSTALL_MODE="source" ;;
      --from-release)   INSTALL_MODE="release" ;;
      --registry=*)     IMAGE_REGISTRY="${arg#*=}" ;;
      -h|--help)      banner; usage; exit 0 ;;
      *)              err "Unknown flag: ${arg}" ;;
    esac
  done
}

usage() {
  echo "Flags:"
  echo "  --lite          Lite edition (OpenVPN, L2TP, basic features)"
  echo "  --full          Full edition (all features, default)"
  echo "  --port=N        Panel HTTPS port (single port; blank/default: 2096)"
  echo "  --home=DIR      Unified data dir for env/certs/db/acme (default: /opt/koris)"
  echo "  --domain=X      Domain name (for SSL)"
  echo "  --ssl=MODE      SSL mode: domain | ip | custom | selfsigned (dev)"
  echo "  --ssl-target=X  Domain or IP for auto SSL (acme.sh)"
  echo "  --cert-path=F   Path to custom cert.pem (used with --ssl=custom)"
  echo "  --key-path=F    Path to custom key.pem (used with --ssl=custom)"
  echo "  --no-knode      Skip knode agent installation"
  echo "  --uninstall     Remove KorisPanel"
  echo "  --version=<tag> Install a specific version tag"
  echo "  --reinstall     Force a clean reinstall"
  echo ""
  echo "URL routing (path-based by default, override for subdomains):"
  echo "  --admin-path=/x/    Admin panel URL prefix (default: /admin/)"
  echo "  --portal-path=/y/   Customer portal URL prefix (default: /account/)"
  echo "  --admin-host=X      Serve admin at subdomain (e.g. admin.example.com)"
  echo "  --portal-host=X     Serve portal at subdomain (e.g. account.example.com)"
  echo ""
  echo "Install method:"
  echo "  --from-release      Use pre-built image from GHCR (default, ~5s)"
  echo "  --from-source       Clone repo and build locally with Docker (~2min)"
  echo "  --registry=X        Custom image registry (default: ghcr.io/anonysec/koris)"
}

prompt_config() {
  # Edition selection
  echo -e "${BOLD}What do you want to install?${NC}"
  echo ""
  echo -e "  ${CYAN}1)${NC} koris      — Full panel (billing, tickets, reseller, all features)"
  echo -e "  ${CYAN}2)${NC} korislite  — Lite panel (OpenVPN, L2TP, users, nodes, settings)"
  echo -e "  ${CYAN}3)${NC} knode      — Node agent only (install on VPN servers)"
  echo ""
  read -rp "$(echo -e "${CYAN}Choose [1/2/3]: ${NC}")" edition_choice </dev/tty
  case "$edition_choice" in
    1) EDITION="full" ;;
    2) EDITION="lite" ;;
    3) EDITION="knode" ;;
    *) err "Invalid choice. Run the script again." ;;
  esac
  echo ""

  # If knode-only, skip panel prompts
  if [[ "${EDITION}" == "knode" ]]; then
    log "Selected: knode (node agent only)"
    return
  fi

  log "Selected: ${EDITION}"
  echo ""

  [[ -z "${DB_PASS}" ]] && DB_PASS="$(gen_secret 16)"

  if [[ -z "${DOMAIN}" ]]; then
    read -rp "$(echo -e "${CYAN}Domain (blank for IP-only): ${NC}")" DOMAIN </dev/tty
  fi
  if [[ "${PANEL_PORT}" == "2096" ]]; then
    read -rp "$(echo -e "${CYAN}Panel port [2096]: ${NC}")" input_port </dev/tty
    PANEL_PORT="${input_port:-2096}"
  fi
  if [[ "${KORIS_HOME}" == "/opt/koris" ]]; then
    read -rp "$(echo -e "${CYAN}Data directory (env/certs/db/acme) [${KORIS_HOME}]: ${NC}")" input_home </dev/tty
    KORIS_HOME="${input_home:-${KORIS_HOME}}"
  fi
  if [[ "${DB_NAME}" == "koris" ]]; then
    read -rp "$(echo -e "${CYAN}DB name [koris]: ${NC}")" input_db </dev/tty
    DB_NAME="${input_db:-koris}"
  fi
  if [[ "${DB_USER}" == "koris" ]]; then
    read -rp "$(echo -e "${CYAN}DB user [koris]: ${NC}")" input_user </dev/tty
    DB_USER="${input_user:-koris}"
  fi

  # SSL mode selection (HTTPS is mandatory — there is no "no SSL" option)
  echo ""
  echo -e "  ${CYAN}1)${NC} Domain — Let's Encrypt via acme.sh (recommended)"
  echo -e "  ${CYAN}2)${NC} IP address — ZeroSSL via acme.sh (works for a bare IP)"
  echo -e "  ${CYAN}3)${NC} Custom cert — provide your own cert.pem + key.pem"
  echo -e "  ${CYAN}4)${NC} Self-signed (DEV ONLY — browser will show a warning)"
  echo ""
  read -rp "$(echo -e "${CYAN}SSL mode [1/2/3/4, default 1]: ${NC}")" ssl_choice </dev/tty
  ssl_choice="${ssl_choice:-1}"
  case "${ssl_choice}" in
    2)
      TLS_MODE="manual"
      setup_acme_ssl "ip"
      ;;
    3)
      TLS_MODE="manual"
      if [[ -z "${SSL_TARGET:-}" ]]; then
        read -rp "$(echo -e "${CYAN}Path to cert.pem: ${NC}")" CERT_PATH </dev/tty
        read -rp "$(echo -e "${CYAN}Path to key.pem: ${NC}")" KEY_PATH </dev/tty
      fi
      [ -f "${CERT_PATH}" ] || err "cert file not found: ${CERT_PATH}"
      [ -f "${KEY_PATH}" ] || err "key file not found: ${KEY_PATH}"
      install_custom_cert
      ;;
    4)
      TLS_MODE="selfsigned"
      ;;
    *)
      TLS_MODE="manual"
      setup_acme_ssl "domain"
      ;;
  esac

  # ─── URL routing ────────────────────────────────────────────────────
  echo ""
  echo -e "${BOLD}How should users reach the admin panel and customer portal?${NC}"
  echo ""
  echo -e "  ${CYAN}1)${NC} Path prefix     — https://${DOMAIN:-<host>}/admin/  and  /account/"
  echo -e "  ${CYAN}2)${NC} Subdomains      — https://admin.${DOMAIN:-<host>}  and  https://account.${DOMAIN:-<host>}"
  echo -e "  ${CYAN}3)${NC} Custom (advanced) — set each path or subdomain individually"
  echo ""
  read -rp "$(echo -e "${CYAN}Choose [1/2/3, default 1]: ${NC}")" url_choice </dev/tty
  case "${url_choice}" in
    2)
      if [[ -z "${DOMAIN}" || "${DOMAIN}" == "_" ]]; then
        warn "Subdomain routing needs a real domain. Falling back to path prefixes."
        ADMIN_PATH="/admin/"
        PORTAL_PATH="/account/"
      else
        ADMIN_HOST="admin.${DOMAIN}"
        PORTAL_HOST="account.${DOMAIN}"
        ADMIN_PATH="/"
        PORTAL_PATH="/"
        echo ""
        warn "You must create these DNS A records pointing to this server:"
        echo -e "    ${YELLOW}admin.${DOMAIN}${NC}    A   $(curl -s https://ifconfig.me 2>/dev/null || echo YOUR_SERVER_IP)"
        echo -e "    ${YELLOW}account.${DOMAIN}${NC}  A   $(curl -s https://ifconfig.me 2>/dev/null || echo YOUR_SERVER_IP)"
      fi
      ;;
    3)
      read -rp "$(echo -e "${CYAN}Admin path prefix [${ADMIN_PATH}]: ${NC}")" in_ap </dev/tty
      ADMIN_PATH="${in_ap:-${ADMIN_PATH}}"
      read -rp "$(echo -e "${CYAN}Portal path prefix [${PORTAL_PATH}]: ${NC}")" in_pp </dev/tty
      PORTAL_PATH="${in_pp:-${PORTAL_PATH}}"
      read -rp "$(echo -e "${CYAN}Admin subdomain (optional, e.g. admin.example.com, blank to keep path routing): ${NC}")" in_ah </dev/tty
      ADMIN_HOST="${in_ah}"
      read -rp "$(echo -e "${CYAN}Portal subdomain (optional): ${NC}")" in_ph </dev/tty
      PORTAL_HOST="${in_ph}"
      # Normalize paths: ensure leading + trailing slash
      [[ "${ADMIN_PATH}"  != /* ]] && ADMIN_PATH="/${ADMIN_PATH}"
      [[ "${ADMIN_PATH}"  != */ ]] && ADMIN_PATH="${ADMIN_PATH}/"
      [[ "${PORTAL_PATH}" != /* ]] && PORTAL_PATH="/${PORTAL_PATH}"
      [[ "${PORTAL_PATH}" != */ ]] && PORTAL_PATH="${PORTAL_PATH}/"
      ;;
    *)
      # Default — leave ADMIN_PATH / PORTAL_PATH at their defaults
      ;;
  esac

  log "Admin  will be served at: ${ADMIN_HOST:+https://${ADMIN_HOST}}${ADMIN_PATH}"
  log "Portal will be served at: ${PORTAL_HOST:+https://${PORTAL_HOST}}${PORTAL_PATH}"
}

# --- Check for existing installation ---
is_existing_installation() {
  [[ -f "${CONFIG_DIR}/panel.env" ]] || return 1
  return 0
}

# --- Clone/fetch source repository ---
clone_source() {
  if [[ -d "${INSTALL_DIR}/.git" ]]; then
    log "Updating source in ${INSTALL_DIR}..."
    git -C "${INSTALL_DIR}" fetch --all --tags --quiet
  else
    log "Cloning panel source..."
    rm -rf "${INSTALL_DIR}"
    git clone "https://github.com/${REPO}.git" "${INSTALL_DIR}" --quiet
  fi

  # Checkout specific version tag if requested
  if [[ -n "${IMAGE_TAG}" ]]; then
    validate_version_tag "${IMAGE_TAG}" "https://github.com/${REPO}.git"
    log "Checking out version: ${IMAGE_TAG}"
    git -C "${INSTALL_DIR}" checkout "${IMAGE_TAG}" --quiet
  else
    # Default: latest main branch
    git -C "${INSTALL_DIR}" checkout main --quiet 2>/dev/null || true
    git -C "${INSTALL_DIR}" pull origin main --quiet 2>/dev/null || true
  fi
}

# --- Create the unified KORIS_HOME (env/certs/db/acme) ---
setup_koris_home() {
  mkdir -p "${KORIS_HOME}"/{certs,acme,data,pgadmin}
  chmod 755 "${KORIS_HOME}"
  # certs/acme must be readable+writeable by the panel container (uid 100 in
  # the panel image), which generates (DEV self-signed) or reads the cert.
  # data must belong to postgres (uid 70 in the Alpine timescaledb image).
  chown -R 100:100 "${KORIS_HOME}"/certs "${KORIS_HOME}"/acme "${KORIS_HOME}"/pgadmin
  chmod 750 "${KORIS_HOME}"/certs "${KORIS_HOME}"/acme
  if docker info &>/dev/null 2>&1; then
    local img
    img=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep -m1 timescale || echo "timescale/timescaledb:latest-pg16")
    docker run --rm -v "${KORIS_HOME}/data:/data" "${img}" chown -R postgres:postgres /data >/dev/null 2>&1 || true
  else
    # No docker yet — best-effort chown to the image's postgres uid (70).
    chown -R 70:70 "${KORIS_HOME}"/data 2>/dev/null || true
  fi
}

# --- Copy a user-supplied cert/key into KORIS_HOME/certs ---
install_custom_cert() {
  setup_koris_home
  mkdir -p "${KORIS_HOME}/certs"
  cp "${CERT_PATH}" "${KORIS_HOME}/certs/cert.pem" || err "failed to copy cert to ${KORIS_HOME}/certs/cert.pem"
  cp "${KEY_PATH}" "${KORIS_HOME}/certs/key.pem" || err "failed to copy key to ${KORIS_HOME}/certs/key.pem"
  chmod 600 "${KORIS_HOME}/certs/key.pem"
  CERT_PATH="${KORIS_HOME}/certs/cert.pem"
  KEY_PATH="${KORIS_HOME}/certs/key.pem"
  log "Custom certificate installed to ${KORIS_HOME}/certs"
}

# --- Obtain a trusted cert via acme.sh (domain=Let's Encrypt, ip=ZeroSSL) ---
setup_acme_ssl() {
  local mode="$1"   # domain | ip
  setup_koris_home
  local target="${SSL_TARGET:-}"
  if [[ -z "${target}" ]]; then
    if [[ "${mode}" == "ip" ]]; then
      read -rp "$(echo -e "${CYAN}Server public IP: ${NC}")" target </dev/tty
    else
      read -rp "$(echo -e "${CYAN}Domain (must point to this server): ${NC}")" target </dev/tty
      DOMAIN="${target}"
    fi
  fi
  [ -n "${target}" ] || err "SSL target (domain or IP) is required"
  [[ "${mode}" == "domain" ]] && DOMAIN="${target}"

  log "Installing acme.sh into ${KORIS_HOME}/acme ..."
  local acme_home="${KORIS_HOME}/acme"
  if [[ ! -x "${acme_home}/acme.sh" ]]; then
    curl -fsS https://get.acme.sh | sh -s -- --home "${acme_home}" --accountemail "admin@${DOMAIN:-koris.local}" \
      || err "acme.sh install failed"
  fi
  local acme="${acme_home}/acme.sh"
  local issuer=""
  [[ "${mode}" == "ip" ]] && issuer="--issuer zerossl"

  log "Issuing certificate for ${target} (${mode}) — ACME HTTP-01 needs :80 reachable now"
  "$acme" --issue ${issuer} -d "${target}" --standalone --keylength ec-256 \
    || err "certificate issuance failed (is :80 reachable and the ${mode} correct?)"

  "$acme" --install-cert -d "${target}" \
    --cert-file "${KORIS_HOME}/certs/cert.pem" \
    --key-file  "${KORIS_HOME}/certs/key.pem" \
    --reloadcmd "true" \
    || err "failed to install certificate into ${KORIS_HOME}/certs"
  chmod 600 "${KORIS_HOME}/certs/key.pem"
  CERT_PATH="${KORIS_HOME}/certs/cert.pem"
  KEY_PATH="${KORIS_HOME}/certs/key.pem"

  # Daily renewal cron (like x-ui)
  ( crontab -l 2>/dev/null | grep -v "acme.sh --cron"; \
    echo "0 3 * * * ${acme} --cron --home ${acme_home} >/dev/null 2>&1" ) | crontab -
  log "Certificate issued and auto-renewal scheduled."
}

# --- Write panel.env configuration ---
write_panel_env() {
  mkdir -p "${CONFIG_DIR}"
  local session_secret setup_key pgadmin_pass radius_secret
  session_secret="$(gen_secret 32)"
  setup_key="$(gen_secret 16)"
  pgadmin_pass="$(gen_secret 8)"
  radius_secret="$(gen_secret 16)"

  CONFIG_DIR="${KORIS_HOME}"
  setup_koris_home

  cat > "${CONFIG_DIR}/panel.env" <<EOF
# KorisPanel Docker Configuration
# Generated by koris.sh — do not edit POSTGRES_PASSWORD manually.
# Every option is documented (commented, with its default) in .env.example;
# this file keeps the required values active so the stack runs out of the box.

# ─── Database (TimescaleDB/PostgreSQL) ────────────────────────────────
PANEL_DB_BACKEND=timescaledb
PANEL_PG_DSN=postgres://${DB_USER}:${DB_PASS}@db:5432/${DB_NAME}?sslmode=disable
POSTGRES_DB=${DB_NAME}
POSTGRES_USER=${DB_USER}
POSTGRES_PASSWORD=${DB_PASS}

# ─── Panel Server ────────────────────────────────────────────────────
# Single port: HTTPS is mandatory; plain HTTP is only a local loopback
# fallback when the certificate can't load. KORIS_HOME is mounted at
# /etc/koris inside the container so certs/config live in one folder.
KORIS_HOME=${KORIS_HOME}
PANEL_ADDR=0.0.0.0:${PANEL_PORT}
PANEL_TLS_ADDR=:${PANEL_PORT}
PANEL_PORT=${PANEL_PORT}
PANEL_SESSION_SECRET=${session_secret}
PANEL_SETUP_KEY=${setup_key}
PANEL_RADIUS_SECRET=${radius_secret}
PANEL_MIGRATIONS=/app/migrations
PANEL_TLS_ENABLED=true
PANEL_TLS_MODE=${TLS_MODE}
# Cert paths are the IN-CONTAINER mount point (/etc/koris == host KORIS_HOME).
PANEL_TLS_CERT=/etc/koris/certs/cert.pem
PANEL_TLS_KEY=/etc/koris/certs/key.pem
PANEL_TLS_CERT_DIR=/etc/koris/certs
PANEL_DOMAIN=${DOMAIN:-}

# ─── URL Routing ─────────────────────────────────────────────────────
# Change ADMIN_PATH / PORTAL_PATH to remap URLs (must start & end with slash).
# Set *_HOST to serve at a subdomain instead; PATH becomes "/" in that case.
PANEL_ADMIN_PATH=${ADMIN_PATH}
PANEL_PORTAL_PATH=${PORTAL_PATH}
PANEL_ADMIN_HOST=${ADMIN_HOST}
PANEL_PORTAL_HOST=${PORTAL_HOST}
# Build-time hint for Vite: makes bundled asset paths match runtime URLs.
# Only used when the Docker image is (re)built locally.
KORIS_ADMIN_BASE=${ADMIN_PATH}
KORIS_PORTAL_BASE=${PORTAL_PATH}

# ─── Build Tags ──────────────────────────────────────────────────────
BUILD_TAGS=${EDITION}

# ─── pgAdmin ─────────────────────────────────────────────────────────
PGADMIN_EMAIL=admin@koris.local
PGADMIN_PASSWORD=${pgadmin_pass}
PGADMIN_PORT=5050
EOF

  # Symlink for docker-compose env_file (some layouts reference it)
  mkdir -p "${INSTALL_DIR}/docker"
  ln -sf "${CONFIG_DIR}/panel.env" "${INSTALL_DIR}/docker/panel.env" 2>/dev/null || true
  log "Configuration written to ${CONFIG_DIR}/panel.env"

  # Compose project .env — docker compose auto-loads this for ${VAR} substitution.
  # The panel vars here are also injected into the container via the service
  # environment: block (which references them), so both substitution and the
  # running container receive the same values.
  cat > "${INSTALL_DIR}/.env" <<EOF
# KorisPanel Docker Compose environment (generated by koris.sh).
# Required values are active; every option's default is documented (commented)
# in .env.example — uncomment and edit there to override.
POSTGRES_DB=${DB_NAME}
POSTGRES_USER=${DB_USER}
POSTGRES_PASSWORD=${DB_PASS}
POSTGRES_PORT=5433

PANEL_PORT=${PANEL_PORT}
PANEL_DOMAIN=${DOMAIN:-localhost}
PANEL_DEV_MODE=false
PANEL_SESSION_SECRET=${session_secret}
PANEL_SETUP_KEY=${setup_key}

# Unified data dir (env, certs, db data, acme) — root + user + docker reachable.
KORIS_HOME=${KORIS_HOME}
PANEL_TLS_ENABLED=true
PANEL_TLS_MODE=${TLS_MODE}
# Cert paths are the IN-CONTAINER mount point (/etc/koris == host KORIS_HOME).
PANEL_TLS_CERT=/etc/koris/certs/cert.pem
PANEL_TLS_KEY=/etc/koris/certs/key.pem
PANEL_TLS_CERT_DIR=/etc/koris/certs

KNODE_API_KEYS=${KNODE_API_KEYS:-}
KNODE_LISTEN_ADDR=0.0.0.0:2083
KNODE_ENABLE_REST=false
KNODE_INSECURE_ALLOW_NO_AUTH=false
KNODE_LOG_LEVEL=info
KNODE_PORT=2083
PGADMIN_PASSWORD=${pgadmin_pass}

# Optional (have sane defaults in docker-compose.yml)
PANEL_ADMIN_PATH=${ADMIN_PATH}
PANEL_PORTAL_PATH=${PORTAL_PATH}
PANEL_ADMIN_HOST=${ADMIN_HOST}
PANEL_PORTAL_HOST=${PORTAL_HOST}
EOF
  log "Compose environment written to ${INSTALL_DIR}/.env"
}

# --- Write version file after successful install ---
write_version_file() {
  local version="${IMAGE_TAG:-}"
  if [[ -z "${version}" ]]; then
    # No explicit tag — read version from VERSION file in source
    version=$(cat "${INSTALL_DIR}/VERSION" 2>/dev/null || echo "latest")
  fi
  mkdir -p "${CONFIG_DIR}"
  echo "${version}" > "${CONFIG_DIR}/version"
  log "Version recorded: ${version}"
}

# --- Docker installation (sole installation path) ---
install_docker() {
  # Ensure Docker is available
  if ! command -v docker &>/dev/null; then
    log "Installing Docker..."
    curl -fsSL https://get.docker.com | sh
  fi
  docker info &>/dev/null || err "Docker installed but daemon is not running"

  # Ensure git is available
  if ! command -v git &>/dev/null; then
    apt-get update -qq && apt-get install -y -qq git >/dev/null 2>&1
  fi

  # Fetch/refresh source. For --from-release mode we still need the docker-compose.yml
  # and migrations — but we skip the pnpm+go build.
  clone_source

  # Write config (skip if reinstalling with existing config)
  if [[ "${FORCE_REINSTALL}" != "yes" ]] || ! is_existing_installation; then
    write_panel_env
  else
    # Reinstall with existing config — source DB_PASS for docker compose
    if [[ -f "${CONFIG_DIR}/panel.env" ]]; then
      DB_PASS=$(grep -oP 'POSTGRES_PASSWORD=\K.*' "${CONFIG_DIR}/panel.env" 2>/dev/null || true)
      if [[ -z "${DB_PASS}" ]]; then
        err "Reinstall failed: POSTGRES_PASSWORD not found in ${CONFIG_DIR}/panel.env"
      fi
      log "Reusing existing configuration from ${CONFIG_DIR}/panel.env"
    fi
  fi

  cd "${INSTALL_DIR}"

  if [[ "${INSTALL_MODE}" == "release" ]]; then
    # Pull pre-built image, no local compile.
    local tag="${IMAGE_TAG:-latest}"
    tag="${tag#v}"  # strip leading v — GHCR tags don't include it
    export KORIS_IMAGE="${IMAGE_REGISTRY}:${tag}"
    log "Pulling ${KORIS_IMAGE}..."
    docker pull "${KORIS_IMAGE}" || err "docker pull failed — falling back to --from-source retries this"
    log "Starting Docker Compose stack..."
    docker compose up -d --pull never || err "Docker Compose failed to start services"
  else
    # Legacy path: build from source. Slower, but works without a release.
    log "Building Docker Compose stack from source (this takes ~2 minutes)..."
    docker compose build || err "Docker build failed — check output above"
    docker compose up -d || err "Docker Compose failed to start services"
  fi

  # Wait for panel to become healthy
  log "Waiting for panel to become healthy..."
  local attempts=0
  while [[ ${attempts} -lt 30 ]]; do
    if docker inspect --format='{{.State.Health.Status}}' panel 2>/dev/null | grep -q "healthy"; then
      log "Panel is healthy"
      write_version_file
      return
    fi
    sleep 2
    attempts=$((attempts + 1))
  done
  warn "Panel did not reach healthy state within 60 seconds — check: docker logs koris"
  # Still write version file — containers are running even if health check timed out
  write_version_file
}

# --- Install knode alongside panel ---
install_knode_docker() {
  log "Installing knode agent on this host..."
  curl -fsSL "https://raw.githubusercontent.com/${KNODE_REPO}/master/install.sh" | bash
}

# --- Clean reinstall (remove containers/images, preserve db-data) ---
clean_reinstall() {
  log "Performing clean reinstall..."
  cd "${INSTALL_DIR}" 2>/dev/null || true
  docker compose down --remove-orphans 2>/dev/null || true
  docker compose rm -f 2>/dev/null || true
  # Remove panel and pgadmin volumes, keep db-data
  docker volume rm koris_panel-data koris_pgadmin-data 2>/dev/null || true
  # Remove project images
  docker images --filter "label=com.docker.compose.project=koris" -q | xargs -r docker rmi -f 2>/dev/null || true
}

# --- Uninstall ---
uninstall() {
  log "Uninstalling KorisPanel..."

  # Stop and remove Docker Compose stack
  if [[ -d "${INSTALL_DIR}" ]]; then
    cd "${INSTALL_DIR}"
    docker compose down -v --remove-orphans 2>/dev/null || true
  fi

  # Remove images
  docker images --filter "label=com.docker.compose.project=koris" -q | xargs -r docker rmi -f 2>/dev/null || true

  # Remove directories
  rm -rf "${INSTALL_DIR}"
  rm -rf "${CONFIG_DIR}"
  rm -f /usr/local/bin/koris

  log "KorisPanel uninstalled"
}

# --- Show installation result ---
show_result() {
  local SERVER_IP
  SERVER_IP=$(curl -fsS4 --max-time 3 https://api.ipify.org 2>/dev/null || hostname -I | awk '{print $1}')

  echo ""
  echo -e "${GREEN}═══════════════════════════════════════${NC}"
  echo -e "${GREEN}  KorisPanel installed successfully!${NC}"
  echo -e "${GREEN}═══════════════════════════════════════${NC}"
  echo ""
  echo -e "  Edition:   ${CYAN}${EDITION}${NC}"
  echo -e "  URL:       ${CYAN}https://${DOMAIN:-${SERVER_IP}}:${PANEL_PORT}${NC}"
  echo -e "  Port:      ${CYAN}${PANEL_PORT}${NC}"
  echo -e "  Config:    ${CONFIG_DIR}/panel.env"
  echo -e "  Source:    ${INSTALL_DIR}"
  echo ""
  echo -e "  ${CYAN}Logs:${NC}      docker compose -f ${INSTALL_DIR}/docker-compose.yml logs -f"
  echo -e "  ${CYAN}Restart:${NC}   docker compose -f ${INSTALL_DIR}/docker-compose.yml restart"
  echo -e "  ${CYAN}Stop:${NC}      docker compose -f ${INSTALL_DIR}/docker-compose.yml down"
  echo ""
  if [[ -f "${CONFIG_DIR}/panel.env" ]]; then
    local setup_key
    setup_key=$(grep -oP 'PANEL_SETUP_KEY=\K.*' "${CONFIG_DIR}/panel.env" 2>/dev/null || echo "")
    if [[ -n "${setup_key}" ]]; then
      echo -e "  ${YELLOW}Setup Key:${NC} ${setup_key}"
      echo -e "  (Use this key on first login to create your admin account)"
      echo ""
    fi
  fi
  echo -e "${GREEN}═══════════════════════════════════════${NC}"
  echo ""
}

# --- Detect existing installations ---
detect_existing() {
  local has_panel="" has_knode="" panel_ver=""

  # Check for existing panel
  if [[ -f "${CONFIG_DIR}/panel.env" ]] || docker ps -a --format '{{.Names}}' 2>/dev/null | grep -qx koris; then
    has_panel="yes"
    panel_ver=$(cat "${INSTALL_DIR}/VERSION" 2>/dev/null || echo "unknown")
  fi

  # Check for existing knode
  if [[ -f "/etc/knode/config.toml" ]] || docker ps -a --format '{{.Names}}' 2>/dev/null | grep -qx knode; then
    has_knode="yes"
  fi

  if [[ -z "${has_panel}" && -z "${has_knode}" ]]; then
    return 1  # No existing installation found
  fi

  # Show what we found
  echo -e "${BOLD}Existing installation detected:${NC}"
  echo ""
  if [[ "${has_panel}" == "yes" ]]; then
    local panel_state
    panel_state=$(docker inspect -f '{{.State.Status}}' koris 2>/dev/null || echo "stopped")
    echo -e "  ${CYAN}●${NC} KorisPanel v${panel_ver} (${panel_state})"
  fi
  if [[ "${has_knode}" == "yes" ]]; then
    local knode_state
    knode_state=$(docker inspect -f '{{.State.Status}}' knode 2>/dev/null || echo "stopped")
    echo -e "  ${CYAN}●${NC} knode (${knode_state})"
  fi
  echo ""

  # Ask what to do
  echo -e "  ${CYAN}1)${NC} Update (pull latest, rebuild — no downtime beyond restart)"
  echo -e "  ${CYAN}2)${NC} Clean reinstall (wipe containers/images, keep DB, rebuild from scratch)"
  echo -e "  ${CYAN}3)${NC} Full wipe & fresh install (removes ALL data including database)"
  echo -e "  ${CYAN}4)${NC} Cancel"
  echo ""
  read -rp "$(echo -e "${CYAN}Choose [1/2/3/4]: ${NC}")" reinstall_choice </dev/tty

  case "${reinstall_choice}" in
    1)
      log "Updating to latest version..."
      cd "${INSTALL_DIR}"
      git fetch origin main --depth=1 >/dev/null 2>&1
      git reset --hard origin/main >/dev/null 2>&1
      docker compose up -d --build
      [[ -f "${INSTALL_DIR}/koris.sh" ]] && cp "${INSTALL_DIR}/koris.sh" /usr/local/bin/koris && chmod +x /usr/local/bin/koris
      log "Updated to v$(cat "${INSTALL_DIR}/VERSION" 2>/dev/null || echo '?')"
      exit 0
      ;;
    2)
      log "Clean reinstall — database data will be preserved"
      FORCE_REINSTALL="yes"
      clean_reinstall
      ;;
    3)
      echo ""
      echo -e "${RED}WARNING: This will delete ALL data including the database.${NC}"
      read -rp "Type 'yes' to confirm: " wipe_confirm </dev/tty
      if [[ "${wipe_confirm}" != "yes" ]]; then
        log "Cancelled."
        exit 0
      fi
      log "Full wipe — removing everything..."
      cd "${INSTALL_DIR}" 2>/dev/null && docker compose down --volumes --remove-orphans 2>/dev/null || true
      docker rm -f koris koris-db koris-pgadmin knode 2>/dev/null || true
      docker volume rm koris_db-data koris_panel-data koris_pgadmin-data 2>/dev/null || true
      docker images --format '{{.ID}} {{.Repository}}' 2>/dev/null | awk '$2 ~ /^koris/ {print $1}' | xargs -r docker rmi -f 2>/dev/null || true
      docker images --filter "label=com.docker.compose.project=koris" -q 2>/dev/null | xargs -r docker rmi -f 2>/dev/null || true
      rm -rf "${INSTALL_DIR}" "${CONFIG_DIR}" /usr/local/bin/koris
      rm -rf /etc/knode
      log "Wipe complete. Starting fresh install..."
      ;;
    4|*)
      log "Cancelled."
      exit 0
      ;;
  esac

  return 0
}

# --- Main ---
do_install() {
  banner
  [[ "$(id -u)" -eq 0 ]] || err "Must run as root"
  detect_os
  parse_args "$@"

  # Handle explicit --reinstall flag (non-interactive, e.g. from koris downgrade)
  if [[ "${FORCE_REINSTALL}" == "yes" ]]; then
    if is_existing_installation; then
      clean_reinstall
    fi
  else
    # Interactive: detect existing installation and ask user what to do
    if detect_existing 2>/dev/null; then
      # User chose option 1 (reinstall) or 2 (wipe) — continue with install
      :
    fi
  fi

  # If knode-only edition was selected, delegate to knode installer
  if [[ "${EDITION}" == "knode" ]]; then
    install_knode_docker
    exit 0
  fi

  # Interactive prompts (skipped if reinstalling with existing config)
  if [[ "${FORCE_REINSTALL}" != "yes" ]]; then
    prompt_config
  fi

  # Docker installation — the only supported path
  install_docker

  # Install CLI management tool
  if [[ -f "${INSTALL_DIR}/koris.sh" ]]; then
    cp "${INSTALL_DIR}/koris.sh" /usr/local/bin/koris
    chmod +x /usr/local/bin/koris
    log "CLI installed: /usr/local/bin/koris"
  fi

  # Optional knode co-installation
  if [[ "${WITH_KNODE}" == "yes" && "${EDITION}" != "knode" ]]; then
    echo ""
    read -rp "$(echo -e "${CYAN}Install knode agent on this server too? [y/N]: ${NC}")" install_knode </dev/tty
    if [[ "${install_knode}" =~ ^[yY] ]]; then
      install_knode_docker
    fi
  fi

  show_result
}

# ═══════════════════════════════════════════════════════════════════════════════
# Dispatcher
# ═══════════════════════════════════════════════════════════════════════════════
case "${1:-}" in
    install)    shift; do_install "$@";;
    start)      cmd_start;; stop) cmd_stop;; restart) cmd_restart;;
    status)     cmd_status;; logs) cmd_logs;; follow|logs-live) cmd_follow;;
    update)     shift; cmd_update "$@";; config) cmd_config;; uninstall) shift; cmd_uninstall "$@";;
    clean)      shift; cmd_clean "$@";;
    db)         shift; cmd_db "$@";;
    pgadmin)    shift; cmd_pgadmin "$@";;
    reinstall)  shift; cmd_reinstall "$@";;
    downgrade)  shift; cmd_downgrade "$@";;
    enable)     docker update --restart unless-stopped koris koris-db koris-pgadmin 2>/dev/null; info "Autostart enabled.";;
    disable)    docker update --restart no koris koris-db koris-pgadmin 2>/dev/null; info "Autostart disabled.";;
    node-status)   echo "Node Agent: $(node_status)";;
    node-restart)  docker restart knode 2>/dev/null && info "Node restarted." || error "Failed to restart knode.";;
    node-logs)     docker logs knode --tail 50 2>/dev/null || error "knode container not found.";;
    help|-h|--help)
        echo "Usage: koris [install|start|stop|restart|status|logs|follow|update|config|uninstall|reinstall|downgrade|clean|db|pgadmin|enable|disable|node-status|node-restart|node-logs]"
        echo "Run without args for the interactive menu.";;
    "")
        if [[ -f "${COMPOSE_FILE}" ]]; then show_menu; else do_install; fi;;
    *) error "Unknown: $1. Run 'koris help'."; exit 1;;
esac
