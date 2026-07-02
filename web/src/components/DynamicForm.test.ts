import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DynamicForm from './DynamicForm.vue'
import type { FormField } from '../types'

const fields:FormField[]=[
  {id:1,key:'name',label:'姓名',type:'text',required:true,placeholder:'你的名字',helpText:'',options:[],position:10,validation:{}},
  {id:2,key:'direction',label:'方向',type:'select',required:true,placeholder:'',helpText:'',options:['开发','设计'],position:20,validation:{}},
  {id:3,key:'avatar',label:'作品图',type:'image',required:false,placeholder:'',helpText:'',options:[],position:30,validation:{}},
]

describe('DynamicForm',()=>{
  it('renders configured fields and required semantics',()=>{
    const wrapper=mount(DynamicForm,{props:{fields,model:{},files:{}}})
    expect(wrapper.text()).toContain('姓名')
    expect(wrapper.find('input[type="text"]').attributes('required')).toBeDefined()
    expect(wrapper.findAll('select option')).toHaveLength(3)
    expect(wrapper.find('input[type="file"]').attributes('accept')).toContain('image/webp')
  })
  it('emits model updates',async()=>{
    const wrapper=mount(DynamicForm,{props:{fields,model:{},files:{}}})
    await wrapper.find('input[type="text"]').setValue('林同学')
    expect(wrapper.emitted('update:model')?.[0]?.[0]).toMatchObject({name:'林同学'})
  })
})

