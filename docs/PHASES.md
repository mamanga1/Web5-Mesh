# 🗺️ Roadmap & Development Phases

XionIA sigue un modelo **Open Core** claro y sostenible: el núcleo y todas las funcionalidades base son 100% open source para fomentar adopción masiva, auditoría comunitaria y soberanía digital. Las funcionalidades avanzadas, de alto rendimiento y enterprise se desarrollan en un repositorio privado bajo licencia comercial.

---

# 🗺️ FASES DEL PROYECTO WEB5-MESH

## ✅ FASE 1 — Fundación Criptográfica (COMPLETADA)

**Features implementadas:**
- ✅ Identidad descentralizada (DID did:maia:*)
- ✅ Criptografía híbrida Ed25519 + X25519
- ✅ ACL (Access Control List) con gestión de confianza
- ✅ Cifrado E2E ChaCha20-Poly1305
- ✅ Noise Protocol IK con Perfect Forward Secrecy (PFS)
- ✅ Sesiones con rekeying automático (cada 5 min)
- ✅ Comandos base: ping, acl, whoami, session
- ✅ Shell interactivo (XION KERNEL)
- ✅ Compilación cross-arch (x86_64 + ARM)

**Entregables técnicos:**
- `src/crypto/identity.go` — Gestión de identidad
- `src/crypto/noise.go` — Handshake Noise IK
- `src/crypto/cipher_session.go` — Sesiones PFS
- `src/crypto/cipher.go` — ChaCha20-Poly1305
- `src/crypto/acl.go` — ACL persistente

---

## 🚧 FASE 2 — Herramienta Soberana para la Comunidad (EN CURSO)

**Objetivo:** Herramienta real y útil para la comunidad. Código libre, gratis para todos.

---

### Features principales:

**Comunicación fácil:**
- ✅ Alias locales para nodos (no pegar DIDs largos)
- 🔲 Chat asíncrono con sesiones (tipo pestañas de navegador)
- 🔲 Notificaciones de mensajes en background
- 🔲 Historial de conversaciones persistente

**Productividad soberana:**
- 🔲 Procesador de texto en el shell (editor mejorado)
- 🔲 Exportar a .pdf con metadatos limpios (sin tracking)
- 🔲 Compartir documentos dentro del Faro/red
- 🔲 Impresora de red (imprimir desde la Jaula)

**Inteligencia Artificial colaborativa:**
- 🔲 MaIA — IA Local-First (inferencia dentro de la red del Faro)
- 🔲 Nodos conectados hacen consultas dentro de la red del Faro
- 🔲 Sin dependencia de APIs externas

**Faro mejorado:**
- 🔲 Puerto dinámico (rota si hay drop)
- 🔲 Evade drops de ISP sin configurar router
- 🔲 Alias y topics configurables

**Shell mejorado:**
- 🔲 Modo "sesión activa" (cuando hacés attach, escribís directo)
- 🔲 Help contextual mejorado
- 🔲 Experiencia tipo terminal Linux
- 🔲 Jaula de Faraday real (cifrado E2E en todo)

---

### Milestones:

**M2.1 — Alias locales y chat asíncrono (2 semanas)**
- Sistema de alias locales en `~/.xion/aliases.json`
- `/chat amigo "hola"` en vez de pegar DID
- Chat abre sesión automáticamente (tipo pestaña)
- Notificaciones en background cuando llega respuesta
- `/session attach 1` para volver al chat sin reescribir comando
- Historial de conversaciones persistente

**M2.2 — Faro con puerto dinámico (2 semanas)**
- Alias y topics configurables (`--alias`, `--topic`)
- Puerto dinámico (rota si hay drop: 42069 → 42070 → 42071...)
- Evade drops de ISP sin configurar router
- Logs con IPs truncadas (privacidad)

**M2.3 — Procesador de texto + PDF (2 semanas)**
- Editor mejorado en el shell
- Exportar a .pdf con metadatos limpios
- Compartir documentos dentro del Faro/red
- Metadatos sin tracking (sin autor, sin software, sin fechas)

**M2.4 — Impresora de red (1 semana)**
- `/printer list` — ver impresoras disponibles
- `/printer set hp` — configurar impresora por defecto
- `/print nota.pdf` — imprimir desde la Jaula
- CUPS integration para Linux

**M2.5 — MaIA Local-First (3 semanas)**
- Integrar modelo de IA en el Faro
- Endpoint para consultas desde nodos
- Inferencia compartida en la red del Faro
- Sin dependencia de APIs externas

**M2.6 — Shell mejorado (2 semanas)**
- Modo "sesión activa" (prompt cambia cuando hacés attach)
- Help contextual mejorado
- Autocompletado básico
- Experiencia pulida tipo terminal Linux

---

### NO hacemos ahora (dejamos para anexo de Fase 2 o Fase 3):

- ❌ Directorio de Faros (`/faro list`) → Sección en iap2p.uk
- ❌ Hosting descentralizado → Anexo Fase 2
- ❌ Multi-conexión a Faros → Anexo Fase 2
- ❌ Modo dialer (conexión saliente) → Fase 3 corporativa
- ❌ Federación de Faros → Fase 3 corporativa
- ❌ Gossip protocol → Fase 3 corporativa

---

### En su lugar:

- ✅ Sección en iap2p.uk donde la gente publica sus Faros manualmente
- ✅ Cada uno monta su Faro (fijo o dinámico)
- ✅ Simple, soberano, sin dependencia central

---

## 🏢 FASE 3 — XION Faraday Suite Enterprise (PRIVADA)

**Estado:** 🔵 Planificación / Desarrollo inicial  
**Inicio estimado:** Q3 2027  
**Modelo:** Licencias, soporte y servicios. Código fuente privado.

### Refactor completo:

**Nueva nomenclatura:**
- 🔲 Renombrar Faro → Bastión
- 🔲 Crear Nación (directorio distribuido)
- 🔲 Modo dual (listener/dialer)
- 🔲 Gossip protocol entre Bastiones
- 🔲 Sistema de alias distribuido global
- 🔲 Bastiones privados/públicos

**Features Enterprise:**
- 🔲 XION Messenger Corporativo con cumplimiento y auditoría
- 🔲 Binarios personalizados + Dead Man's Switch
- 🔲 Jaula de Faraday Enterprise + dashboards
- 🔲 Clustering, alta disponibilidad y SLAs
- 🔲 Soporte 24/7 y consultoría
- 🔲 Anti-triangulación y protecciones avanzadas

**Features técnicas avanzadas:**
- 🔲 Auto-discovery de peers (mDNS + DHT)
- 🔲 Mesh routing con propagación gossip
- 🔲 Replicación eventual de Bastiones entre nodos
- 🔲 Quorum de validación para mensajes críticos
- 🔲 Rate limiting y anti-spam por peer
- 🔲 Health checks entre nodos federados
- 🔲 Métricas Prometheus + dashboard
- 🔲 Logs estructurados (JSON) con rotación
- 🔲 Auto-healing: reconexión automática ante fallos
- 🔲 Backpressure y flow control en congestión
- 🔲 Compresión de payloads (zstd)
- 🔲 Deduplicación de mensajes por hash
- 🔲 Time-to-live (TTL) configurable por mensaje

---

## 📊 Estado actual

| Fase | Estado | Progreso |
|------|--------|----------|
| 1 — Fundación Criptográfica | ✅ Completada | 100% |
| 2 — Herramienta Soberana | 🚧 En curso | ~15% |
| 3 — XION Faraday Suite Enterprise | 🔵 Planificada | 0% |

---

## ⚠️ Riesgos y Mitigación

| Riesgo | Mitigación |
|:-------|:-----------|
| Dependencia de hardware específico | Stack optimizado para ARM64, también corre en x86 |
| Adopción lenta | Estrategia de comunidades locales + documentación bilingüe + sección en iap2p.uk para Faros |
| Seguridad en IA local | MaIA corre dentro de la Jaula de Faraday (datos nunca salen del nodo) |
| Sostenibilidad post-grant | Modelo de licencias premium (Fase 3 Enterprise) + soporte corporativo |
| Chat asíncrono con sesiones | Diseño incremental: primero alias, luego sesiones, luego modo attach |
| PDF/impresora sin exec.Command | Librerías nativas en Go (go-fitz para PDF, IPP para impresoras) |
| MaIA en hardware limitado | Modelos cuantizados (GGUF) + fallback a inferencia mínima |
| Puerto dinámico del Faro | Rango 42069-42169 con rotación automática + logs con IPs truncadas |

---

## 💰 Solicitud de Financiamiento – Fase 2

Buscamos **$62,000 USD** para financiar los próximos 12 meses de desarrollo full-time de la Fase 2.

### Desglose detallado del presupuesto:

| Categoría                        | Monto (USD) | Porcentaje | Detalle |
|:---------------------------------|------------:|-----------:|:--------|
| **Desarrollo Full-Time (2 devs)**| $36,000    | 58%       | Salarios + equipo core |
| **Hardware & Testing en Campo**  | $9,000     | 15%       | TV Boxes, Raspberry, movilidad NEA |
| **Auditorías de Seguridad**      | $8,000     | 13%       | Revisión externa cripto y protocolo |
| **Hosting / Servidores semilla** | $4,000     | 6%        | Faros públicos y monitoreo |
| **Legal, contabilidad y misc.**  | $5,000     | 8%        | Estructura legal + gastos operativos |

**Total solicitado:** **$62,000 USD**

### Uso de los fondos:
Este financiamiento nos permitirá pasar de un **PoC hiperfuncional ya probado en campo** a una plataforma **madura, estable y lista para adopción masiva** en entornos reales de baja conectividad.

---

## 🔮 Sostenibilidad Post-Grant

Una vez finalizada la Fase 2, el proyecto se sostendrá mediante licencias comerciales de la Fase 3, soporte empresarial, hosting gestionado y contribuciones de la comunidad.

> *"Construimos la base en público para que el mundo confíe. Construimos el futuro en privado para que el proyecto sobreviva."*
>
> **Última actualización:** 29 de junio de 2026

