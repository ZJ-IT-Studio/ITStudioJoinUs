import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import UiDialog from './UiDialog.vue'
import UiSelect from './UiSelect.vue'
import UiSwitch from './UiSwitch.vue'

afterEach(()=>{ document.body.innerHTML='' })

describe('admin Reka UI primitives',()=>{
  it('renders the selected option with accessible trigger',()=>{
    const wrapper=mount(UiSelect,{props:{modelValue:'owner',ariaLabel:'账号权限',options:[{label:'可编辑',value:'owner'},{label:'仅查看',value:'readonly'}]}})
    expect(wrapper.get('button').attributes('aria-label')).toBe('账号权限')
    expect(wrapper.text()).toContain('可编辑')
  })

  it('exposes switch state',()=>{
    const wrapper=mount(UiSwitch,{props:{modelValue:true,label:'账号启用状态'}})
    expect(wrapper.get('button').attributes('data-state')).toBe('checked')
  })

  it('portals an accessible dialog',async()=>{
    const wrapper=mount(UiDialog,{attachTo:document.body,props:{open:true,title:'批次设置',description:'编辑招新批次'}})
    await new Promise(resolve=>setTimeout(resolve,0))
    expect(document.body.textContent).toContain('批次设置')
    expect(document.body.textContent).toContain('编辑招新批次')
    wrapper.unmount()
  })
})

