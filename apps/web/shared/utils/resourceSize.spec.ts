import { describe, expect, it } from 'vitest'
import {
  applyResourceSizeInput,
  clampResourceSizeAmount,
  detectResourceSizeUnit,
  joinResourceSize,
  parseResourceSize,
  splitResourceSize
} from './resourceSize'

describe('parseResourceSize', () => {
  it('accepts a number plus MB or GB', () => {
    expect(parseResourceSize('15 MB')).toBe('15 MB')
    expect(parseResourceSize('3.8gb')).toBe('3.8 GB')
    expect(parseResourceSize('1GB')).toBe('1 GB')
    expect(parseResourceSize('1.50 GB')).toBe('1.5 GB')
  })

  it('rejects everything else', () => {
    expect(parseResourceSize('')).toBeNull()
    expect(parseResourceSize('820 KB')).toBeNull()
    expect(parseResourceSize('2GB【模拟器可玩】')).toBeNull()
    expect(parseResourceSize('15 MB 左右')).toBeNull()
  })
})

describe('splitResourceSize', () => {
  it('prefills from a dirty historical value', () => {
    expect(splitResourceSize('【PC+安卓直装】23.43GB')).toEqual({
      amount: '23.43',
      unit: 'GB'
    })
    expect(splitResourceSize('2GB【模拟器可玩】')).toEqual({
      amount: '2',
      unit: 'GB'
    })
  })

  it('defaults an empty field to GB', () => {
    expect(splitResourceSize('')).toEqual({ amount: '', unit: 'GB' })
  })
})

describe('joinResourceSize and clampResourceSizeAmount', () => {
  it('joins a filled amount', () => {
    expect(joinResourceSize({ amount: '2', unit: 'GB' })).toBe('2 GB')
    expect(joinResourceSize({ amount: '', unit: 'MB' })).toBe('')
  })

  it('keeps only digits and two decimals', () => {
    expect(clampResourceSizeAmount('12a.3b45')).toBe('12.34')
    expect(clampResourceSizeAmount('2GB')).toBe('2')
    expect(clampResourceSizeAmount('3,8')).toBe('3.8')
  })
})

describe('detectResourceSizeUnit', () => {
  it('reads a unit typed after the number', () => {
    expect(detectResourceSizeUnit('500MB')).toBe('MB')
    expect(detectResourceSizeUnit('500mb')).toBe('MB')
    expect(detectResourceSizeUnit('500 m')).toBe('MB')
    expect(detectResourceSizeUnit('3.8GB')).toBe('GB')
    expect(detectResourceSizeUnit('2G')).toBe('GB')
    expect(detectResourceSizeUnit('15M')).toBe('MB')
  })

  it('ignores a bare number', () => {
    expect(detectResourceSizeUnit('500')).toBeNull()
    expect(detectResourceSizeUnit('3.8')).toBeNull()
    expect(detectResourceSizeUnit('')).toBeNull()
  })
})

describe('applyResourceSizeInput', () => {
  it('lets a typed unit override the selected unit', () => {
    expect(applyResourceSizeInput('500MB', 'GB')).toEqual({
      amount: '500',
      unit: 'MB'
    })
    expect(applyResourceSizeInput('3.8gb', 'MB')).toEqual({
      amount: '3.8',
      unit: 'GB'
    })
    expect(applyResourceSizeInput('15m', 'GB')).toEqual({
      amount: '15',
      unit: 'MB'
    })
  })

  it('keeps the selected unit when the amount has no unit', () => {
    expect(applyResourceSizeInput('500', 'GB')).toEqual({
      amount: '500',
      unit: 'GB'
    })
    expect(applyResourceSizeInput('3.8', 'MB')).toEqual({
      amount: '3.8',
      unit: 'MB'
    })
  })
})
