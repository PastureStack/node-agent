# Node 0.13.35 / Host API 0.38.9 candidate

Linux Node requires the exact independently packaged Host API 0.38.9 producer.
The normal configcontent sequence installs Host API before Node. An old or dirty
producer fails closed; no embedded legacy fallback is introduced. Host API owns
stream authorization, EOF drain and durable terminal evidence. Node only verifies
and supervises its producer, forwards current identity in memory and binds its
lifetime to Node.

Node's Go tar/Windows zip and optional Docker Hub runtime image share 0.13.35.
The optional Node release can also publish its verified Go image to GHCR under
v0.13.35. These Go images are distinct from the Server repository's Python
bootstrap compatibility image (Server 583 selects v1.2.34; standalone Engine
defaults still select pinned v1.2.31). This pairing does not rebuild either
bootstrap image. Publication and
matching Server native acceptance remain pending; no unreleased digest is known.
The existing independently pinned Server Windows producer is not changed by this
Linux Host API pairing.
