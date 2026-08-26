# AI-assisted conversation drafts — RentStage v0.20.0

## Purpose

RentStage v0.20.0 introduces optional AI generation for private web-chat response drafts. It improves the operator's starting point without allowing a model to contact a customer or change rental operations autonomously.

The feature is intentionally a drafting system, not an autonomous customer-service agent.

## Trust boundary

1. A visitor submits an inbound message through the first-party public web chat.
2. RentStage validates and persists the inbound message idempotently.
3. The configured draft provider generates a proposed response.
4. RentStage validates the provider result and stores it as an outbound assistant `DRAFT`.
5. The visitor cannot retrieve the draft.
6. An authenticated team member reviews, optionally edits, and explicitly publishes the response.
7. Only the published `SENT` message becomes visible in the visitor session.

No model call approves a quote, reserves inventory, changes availability, creates a fiscal document, or sends a Meta message.

## Provider modes

### Deterministic rules

`ASSISTANT_AI_MODE=rules` is the default. It uses bounded Spanish templates and requires no external provider, credential, network call, or cloud cost.

Metadata:

```text
engine=WEB_CHAT_RULES
model=DETERMINISTIC_V1
used_fallback=false
human_approval_required=true
```

### Vertex AI

`ASSISTANT_AI_MODE=vertex` enables the Vertex AI adapter. The default model is `gemini-2.5-flash`, with an eight-second timeout and a 512-output-token limit.

The draft request includes the tenant identity, visitor name, draft kind, and current customer message. v0.20.0 does not provide full conversation history, inventory retrieval, pricing tools, quote mutation, or long-term memory to the model.

Successful metadata:

```text
engine=VERTEX_AI
model=<configured model>
used_fallback=false
human_approval_required=true
```

## Safe fallback

Provider errors, timeouts, empty output, missing engine/model metadata, and output beyond the application message limit are rejected. RentStage then generates a deterministic draft and records:

```text
engine=WEB_CHAT_RULES
model=DETERMINISTIC_V1
used_fallback=true
human_approval_required=true
```

Fallback affects only the proposed draft. It does not bypass human review or publish automatically.

## Idempotency

Every visitor submission carries a client-generated message identifier. Initial and follow-up processing associate the outbound draft with that source identifier.

Repeating the same accepted identifier returns the existing session state instead of inserting another inbound message or generating another assistant draft. This prevents network retries from multiplying model calls or reviewer work.

## Provenance and auditability

Draft metadata is stored in the existing `assistant_messages.metadata` JSON field; no schema migration is required.

Relevant fields include:

- `engine`;
- `model`;
- `used_fallback`;
- `human_approval_required`;
- `source_message_id`.

The assistant inbox translates these values into reviewer-facing provenance badges while the message remains a pending assistant draft. After publication, the response is represented as a human-sent message; the original generation metadata remains available for audit in persistence.

## Credentials and IAM

Local development should use Application Default Credentials with service-account impersonation. Do not create, distribute, or commit long-lived service-account JSON keys.

Cloud Run uses its attached API runtime service account. The runtime requires `roles/aiplatform.user`; project policy may also require `roles/serviceusage.serviceUsageConsumer`.

The developer performing local impersonation requires `roles/iam.serviceAccountTokenCreator` on the target service account. ADC files and temporary Compose overrides must stay outside Git and be mounted read-only only for the duration of a local Vertex test.

## Runtime defaults

| Setting | Default |
| --- | --- |
| Mode | `rules` |
| Location | `us-central1` |
| Model | `gemini-2.5-flash` |
| Timeout | `8s` |
| Maximum output | `512` tokens |

The safe `rules` default applies to Docker Compose and staging unless explicitly overridden.

## Privacy considerations

Enabling Vertex transmits the configured draft request context to the selected Google Cloud project and region. Operators must use an approved project, IAM boundary, retention policy, and customer disclosure appropriate to their jurisdiction and business policy.

Secrets, raw web-chat session tokens, authentication cookies, database credentials, and unrelated tenant data are not part of the draft request.

## Deferred scope

The following remain outside v0.20.0:

- automatic publication or autonomous customer contact;
- full conversation-history prompting;
- retrieval-augmented generation over catalog or inventory data;
- model-created quotes, reservations, payments, or DTE operations;
- real Meta, Instagram, Facebook, or WhatsApp AI delivery;
- streaming responses, tool calling, or model-managed memory.

These capabilities require separate authorization, privacy, safety, evaluation, and rollback milestones.
