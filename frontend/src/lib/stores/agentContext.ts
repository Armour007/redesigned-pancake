// frontend/src/lib/stores/agentContext.ts
import { writable } from 'svelte/store';

// Define the shape of the data we want to store
interface AgentContext {
    agentId: string | null;
    organizationId: string | null;
}

// Create a writable store with initial null values
export const agentContext = writable<AgentContext>({
    agentId: null,
    organizationId: null,
});