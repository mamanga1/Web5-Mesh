# XionIA - Documentación Técnica (v1.0)

## 1. Arquitectura de Red

XionIA no es P2P en el sentido clásico (Kademlia/DHT). Es una red de túneles cifrados punto a punto donde los nodos se conectan a un **Relay Ciego** para superar NATs y firewalls.

### 1.1. Componentes

| Componente | Función | Persistencia |
| :--- | :--- | :--- |
| **Nodo (Mesh)** | Cliente que se conecta al Faro y a otros nodos. | Almacena su identidad y ACL en `~/.xion/`. |
| **Faro (Relay)** | Retransmite paquetes entre nodos. **No descifra** el contenido. | **Volátil** (solo RAM). No guarda logs ni estados. |
| **Jaula de Faraday** | Entorno aislado del nodo. Archivos, configuraciones y workspace. | Persistente en `~/.xion/workspace/`. |

### 1.2. Flujo de Comunicación (UDP y WebSocket)

El Faro soporta dos protocolos de transporte para adaptarse a diferentes entornos de red:

1.  **UDP (Puerto 54321):** Protocolo nativo, rápido y eficiente. Es la vía preferida.
2.  **WebSocket (Puerto 443):** Sobre TCP/TLS. Diseñado para atravesar firewalls corporativos y redes que bloquean UDP, aprovechando que el tráfico HTTPS (443) suele estar permitido.

**El cliente detecta automáticamente el protocolo según el puerto configurado en `FARO_ADDR`.**

*Nota sobre CGNAT:* Ni UDP ni WebSocket "saltan" el CGNAT por sí solos. Para nodos detrás de CGNAT, es necesario que el **Faro tenga una IP Pública** y los nodos se conecten a él de forma **saliente**.

### 1.3. El Faro como Relay Ciego

El Faro solo entiende comandos de enrutamiento. No procesa el payload.

1.  **ANNOUNCE:** El nodo se registra en el Faro (DID → IP:Puerto).
2.  **RELAY:** El nodo A envía un paquete cifrado para el nodo B. El Faro reenvía el paquete a B y devuelve un `ACK` a A.
3.  **RESPONSE:** El nodo B responde a A a través del Faro.

---

## 2. Stack Criptográfico

| Propósito | Algoritmo | Uso |
| :--- | :--- | :--- |
| **Identidad y Firmas** | Ed25519 | Generación de DIDs (`did:maia:...`) y firma de mensajes/archivos. |
| **Intercambio de Claves** | X25519 | Derivación de secreto compartido (ECDH) para sesiones E2E. |
| **Cifrado Simétrico** | ChaCha20-Poly1305 | Cifrado de mensajes y archivos (AEAD). |
| **Handshake** | Noise Protocol IK | Establecimiento de sesiones con Perfect Forward Secrecy (PFS). |
| **Anti-DPI** | Padding Aleatorio | Se añaden 50-200 bytes aleatorios a cada paquete UDP para ofuscar el tráfico. |

---

## 3. Seguridad y Privacidad

*   **Zero-Knowledge:** El Faro no tiene acceso a las claves privadas. Solo retransmite datos cifrados.
*   **Hostil por Diseño:** El sistema operativo del host es considerado un entorno hostil. Por eso, toda la actividad sensible del nodo ocurre dentro de la **Jaula de Faraday** (`~/.xion/workspace/`), con permisos `0600`.
*   **ACL (Lista de Control de Acceso):** La comunicación solo se establece con nodos incluidos explícitamente en la `acl.json` del usuario.
*   **Logs Truncados:** El Faro nunca registra IPs completas en los logs para preservar la privacidad de los nodos.
*   **ACK (Confirmación de Entrega):** El Faro confirma al emisor que el mensaje fue reenviado al destino, garantizando la fiabilidad del relay.

---

## 4. Estructura de Directorios del Nodo

```
~/.xion/                       # Jaula de Faraday del Nodo
├── node.key                   # Clave privada (NUNCA COMPARTIR)
├── acl.json                   # Lista de nodos de confianza
├── aliases.json               # Alias locales para DIDs
├── workspace/                 # Archivos de usuario (aislados del host)
│   ├── archivo.pdf            # Permisos 0600 (solo el usuario)
│   └── archivo.pdf.sig        # Firma del archivo
└── sessions/                  # Historial de chats (persistente)
```

---

## 5. Especificación Técnica de Comandos (Core)

*(Nota: Para la lista completa y ejemplos de uso, ver [COMMANDS.md](COMMANDS.md))*

*   `whoami`: Muestra el DID y claves públicas del nodo.
*   `acl`: Gestiona la lista de nodos de confianza (add, remove, list, import, clear).
*   `alias`: Gestiona alias locales (add, remove, list).
*   `chat`: Envía un mensaje E2E a un alias o DID.
*   `group`: Gestiona grupos de comunicación (create, add, send, etc.).
*   `faro`: Configura o resetea la dirección del Faro (`set`, `reset`).
*   `import` / `export`: Mueve archivos entre el host y la Jaula de Faraday.
*   `sign` / `verify`: Firma y verifica archivos con Ed25519.
*   `clear --force`: Borra la identidad y la ACL del nodo.

---

## 6. Guía Rápida de Uso

### 6.1. Levantar un Faro

```bash
# 1. Compilar
go build -trimpath -ldflags="-s -w" -o faro ./cmd/faro

# 2. (Opcional) Generar certificados TLS para WebSocket (443)
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes -subj "/CN=faro.local"

# 3. Ejecutar
nohup ./faro > faro.log 2>&1 &
```

### 6.2. Conectar un Nodo

```bash
# 1. Compilar
go build -o mesh ./cmd/mesh

# 2. Configurar el Faro (en el nodo cliente)
export FARO_ADDR="<IP_DEL_FARO>:54321"  # Para UDP
# o
export FARO_ADDR="<IP_DEL_FARO>:443"     # Para WebSocket (TLS)

# 3. Iniciar la shell
./mesh shell
```

---

## 7. Roadmap Técnico

El desarrollo está organizado por fases. Para ver el estado actual y el roadmap completo, consultar [PHASES.md](PHASES.md).

*   **Fase 1 (✅ Completada):** Fundación Criptográfica, Faro Dual (UDP/WS), ACL, Jaula de Faraday.
*   **Fase 2 (🚧 En curso):** Herramienta Soberana (Chat asíncrono, IA Local, Impresora, Puerto Dinámico).
*   **Fase 3 (🔵 Planificada):** XION Faraday Suite Enterprise (Federación, Clustering, Gobernanza).

---

*Documentación técnica de XionIA - La Xión Digital 🦾*
*Última actualización: 10 de Julio de 2026*
