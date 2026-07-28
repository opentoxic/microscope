import { ref } from 'vue'
import type { EntryType } from './types'

/** Entry type for the open detail page; drives sidebar active state. */
export const activeDetailEntryType = ref<EntryType | ''>('')
