export interface ContentCard { label: string; title: string; body: string }
export interface SiteContent {
  heroEyebrow: string; heroTitle: string; heroSubtitle: string
  manifestoTitle: string; manifestoBody: string; directionsTitle: string
  directions: ContentCard[]; values: ContentCard[]; process: ContentCard[]; faqs: ContentCard[]
  contactTitle: string; contactBody: string; contactLink: string
}
export interface Campaign { id: number; name: string; slug: string; status: 'draft'|'open'|'closed'|'archived'; startsAt?: string; endsAt?: string; formLocked: boolean }
export type FieldType = 'text'|'textarea'|'number'|'date'|'url'|'radio'|'checkbox'|'select'|'image'
export interface FormField { id: number; campaignId?: number; key: string; label: string; type: FieldType; required: boolean; placeholder: string; helpText: string; options: string[]; position: number; validation: Record<string, unknown> }
export interface ReviewStatus { id: number; name: string; color: string; description: string; position: number; isDefault: boolean }
export interface Application { id: number; campaignId: number; studentId: string; email: string; systemStatus: 'submitted'|'withdrawn'; revision: number; submittedAt: string; reviewStatus?: ReviewStatus }
export interface ApplicationDetail { application: Application; answers: { key:string; label:string; type:string; value:unknown }[]; uploads:{id:number;key:string;label:string;name:string;mime:string;size:number}[]; campaign:Campaign; canWithdraw:boolean; canResubmit:boolean; notes?:unknown[]; history?:unknown[] }

