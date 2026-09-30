# Copyright 2026 Scaleway
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
from __future__ import annotations

from abc import ABC, abstractmethod

from qall_registry_client.v1.objects import Block, Tag


class RegistryClient(ABC):
    @abstractmethod
    def check(self, hashes: list[str]) -> list[str]:
        """Checks which of the given hashes are missing on the server."""
        raise NotImplementedError

    @abstractmethod
    def push(self, blocks: list[Block]) -> int:
        """Pushes the given blocks to the server. Returns the count of actually written blocks."""
        raise NotImplementedError

    @abstractmethod
    def tag(self, tag: Tag) -> str:
        """Tags a workflow with a human-readable name and version. Returns the assigned URI."""
        raise NotImplementedError

    @abstractmethod
    def untag(self, name: str, version: str) -> bool:
        """Removes the tag with the given name and version. Returns whether the tag was found and removed."""
        raise NotImplementedError

    @abstractmethod
    def list_tags(self, name: str) -> list[Tag]:
        """Lists all tags with the given name."""
        raise NotImplementedError

    @abstractmethod
    def resolve(self, name: str, version: str) -> str:
        """Resolves the given name and version to a workflow URI. Returns the URI if found, or raises if not found."""
        raise NotImplementedError
