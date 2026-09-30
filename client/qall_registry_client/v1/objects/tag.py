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

from dataclasses import dataclass


@dataclass
class Tag:
    """
    Named, versioned reference to a computation graph.
    The only mutable object in the registry — a tag can be moved.
    Does not duplicate data: only references the root node hash.
    """

    name: str
    version: str
    root_hash: str

    def to_dict(self) -> dict:
        return {
            "name": self.name,
            "version": self.version,
            "root_hash": self.root_hash,
        }

    @property
    def uri(self) -> str:
        return f"{self.name}:{self.version}"
