# XionIA - Jaula de Faraday (Bóveda Soberana)

## 🛡️ El Principio Fundamental

**El host es un medio hostil.** Todo lo que toca el sistema operativo puede estar comprometido: malware, rootkits, backdoors. Por eso XionIA crea su propio espacio aislado: la **Jaula de Faraday** (`~/.u2p/workspace/`).

```
┌─────────────────────────────────────────────┐
│  HOST (Medio Hostil)                        │
│  ┌───────────────────────────────────────┐  │
│  │  ~/.u2p/workspace/ (Jaula de Faraday) │  │
│  │  ├── archivo.pdf (0600)               │  │
│  │  ├── archivo.pdf.sig (firma)          │  │
│  │  └── ...                              │  │
│  └───────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

## 🔄 Flujo Completo: Import → Sign → Verify → Export

### Paso 1: IMPORT - Meter archivo del host a la bóveda

```
xion@nodo:~$ import ~/documento.pdf
✅ Archivo ingresado a la bóveda:
   ├── Origen: /home/usuario/documento.pdf
   ├── Destino: ~/.u2p/workspace/documento.pdf
   └── Permisos: 0600 (solo tú)
```

### Paso 2: SIGN - Firmar criptográficamente

```
xion@nodo:~$ sign documento.pdf
✅ Archivo firmado criptográficamente:
   ├── Archivo: documento.pdf (23 bytes)
   ├── Hash SHA256: 2ee498c8fa0f778e...
   ├── Firma: documento.pdf.sig
   ├── Firmante: did:maia:5XVNWhUtMNH...
   └── Timestamp: 2026-06-17 11:21:15
```

### Paso 3: VERIFY - Verificar integridad y autenticidad

```
xion@nodo:~$ verify documento.pdf
✅ VERIFICACIÓN EXITOSA:
   ├── Integridad: ✅ Hash válido
   ├── Autenticidad: ✅ Firma válida
   ├── Firmante: Tú mismo
   └── El archivo es auténtico y no fue modificado.
```

### Paso 4: EXPORT - Sacar archivo de la bóveda

```
xion@nodo:~$ export documento.pdf ~/Desktop/
✅ Archivo exportado de la bóveda:
   ├── Origen (Bóveda): ~/.u2p/workspace/documento.pdf
   ├── Destino (Host): ~/Desktop/documento.pdf
   └── Permisos: 0644 (listo para compartir)
```

## 📤 Transporte Agnóstico

Una vez exportados, podés enviar el archivo y su firma por **cualquier medio** (Email, WhatsApp, USB, Web). El receptor solo necesita hacer `import` de ambos y ejecutar `verify` para garantizar matemáticamente la integridad y autenticidad.

## 🛠️ Comandos Unix en la Bóveda

Todos estos comandos operan **dentro de la Jaula de Faraday**:

| Comando | Descripción |
|:--------|:------------|
| `/ls` | Listar contenido |
| `/cat <archivo>` | Ver contenido |
| `/rm <archivo>` | Borrar archivo |
| `/mv <origen> <destino>` | Mover/renombrar |
| `/cp <origen> <destino>` | Copiar archivo |
| `/mkdir <carpeta>` | Crear carpeta |
| `/edit <archivo>` | Editor integrado (`:wq` para guardar) |

## 💡 Casos de Uso

### Caso 1: Firmar un contrato

```
import ~/contrato.pdf
sign contrato.pdf
verify contrato.pdf
export contrato.pdf ~/Desktop/
export contrato.pdf.sig ~/Desktop/
# Enviá ambos archivos por email al receptor
```

### Caso 2: Detectar manipulación

```
# Alguien modificó el archivo en el camino
verify contrato.pdf
❌ INTEGRIDAD COMPROMETIDA:
   ├── Hash esperado: 2ee498c8fa0f778e...
   ├── Hash actual: 9f8e7d6c5b4a3210...
   └── El archivo fue modificado después de la firma.
```

---

*Guía de la Jaula de Faraday de XionIA - La Xión Digital 🦾*
```
