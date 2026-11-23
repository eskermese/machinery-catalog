import type { Dictionary } from './ru'

/**
 * Английский словарь.
 *
 * Тип обязывает повторить каждый ключ русского: забытая строка — ошибка
 * сборки, а не пустое место на странице.
 */
export const en: Dictionary = {
  shell: {
    machinery: 'Machinery',
    menu: 'Menu',
    siteMenu: 'Site menu',
    siteSections: 'Site sections',
    catalogSections: 'Catalogue sections',
    wholeSection: 'All machines in this section',
    searchMachinery: 'Search machinery',
    language: 'Site language',
    tagline: 'Machinery, spare parts and service across Kazakhstan.',
    sections: 'Sections',
    getInTouch: 'Get in touch',
    leaveRequest: 'Request a call',
    address: 'Address',
    serviceAndWarehouse: 'Service centre and parts warehouse are at the same address',
  },

  siteSections: {
    spareParts: 'Spare parts',
    service: 'Service',
    leasing: 'Leasing',
    about: 'About us',
    contacts: 'Contacts',
  },

  machine: {
    availability: (value: string) =>
      ({ in_stock: 'In stock', on_order: 'To order' })[value] ?? value,
    condition: (value: string) => ({ new: 'New', used: 'Used' })[value] ?? value,
    weeks: (count: number) => `${count} ${count === 1 ? 'week' : 'weeks'}`,
    yes: 'Yes',
    no: 'No',
    metaDescription: (brand: string, model: string, condition: string, availability: string) =>
      `${brand} ${model} — ${condition.toLowerCase()}, ${availability.toLowerCase()}. ` +
      'Prices and specifications in the Techline catalogue.',
    atRate: (date: string) => `at the rate on ${date}`,
    priceOnRequest: 'Price on request',
    priceOnRequestHint: ': leave a request to find out the price',
  },

  searchPage: {
    title: 'Search',
    titleWith: (query: string) => `Search: ${query}`,
    heading: (query: string) => `Search: “${query}”`,
    failed: 'Search failed',
    sectionsByQuery: 'Sections matching the query',
    catalogSections: 'Catalogue sections',
    wholeSectionWithFilters: 'Whole section with filters',
    otherGroup: 'Other',
    filtersNote:
      'Specification filters work inside a section: search results span every section at once, and “bucket capacity” on an excavator and on a loader are different specifications.',
    askTitle: 'What are you looking for?',
    askText: 'Enter a name, model or machine type — for example “excavator” or “XE370G”.',
    toHome: 'Go to the home page',
    openCatalog: 'Open the catalogue',
    sectionOnlyTitle: 'No machines with that name',
    sectionOnlyText: (query: string) =>
      `Search looks at the name, brand and model of a machine. None is called “${query}” — but a catalogue section is: it is in the list above.`,
    emptyTitle: (query: string) => `Nothing found for “${query}”`,
    emptyWithFilters:
      'It may be the filters: try dropping some conditions. Search looks at the name, brand and model — not at specifications or descriptions.',
    emptyPlain:
      'Search looks at the name, brand and model — not at specifications or descriptions. Try something shorter: “excavator” instead of “crawler excavator XCMG”.',
  },

  suggest: {
    heroPlaceholder: 'Excavator, XE370G, loader…',
    headerPlaceholder: 'Search machinery',
    label: 'Search machinery',
    listLabel: 'Search suggestions',
    machines: 'Machines',
    sectionTag: 'section',
    showAll: (query: string) => `Show all results for “${query}”`,
    emptyTitle: (query: string) => `Nothing found for “${query}”`,
    emptyText:
      'Search looks at the name, brand and model. Try something shorter — “excavator” instead of “crawler excavator XCMG” — or pick a section in the catalogue.',
    openCatalog: 'Open the catalogue',
  },

  comparePage: {
    title: 'Comparison',
    metaDescription: 'Compare specifications of machinery from the Techline catalogue.',
    emptyTitle: 'Nothing to compare',
    needSecondTitle: 'At least a second machine is needed',
    emptyText:
      'Mark two or three machines with the “Compare” button in the catalogue or on a machine page — the specification table will appear here.',
    toCatalog: 'To the catalogue',
    dashNote:
      'A dash means the specification is not filled in for this machine, not that it has none.',
    attribute: 'Specification',
    remove: 'Remove',
    removeHint: (name: string) => `Remove “${name}” from the comparison`,
    hideSame: (count: number) => `Hide identical (${count})`,
    showSame: (count: number) => `Show identical (${count})`,
    allSame: 'These machines do not differ in the specifications that are filled in.',
  },

  home: {
    title: 'Machinery catalogue',
    metaDescription:
      'Excavators, loaders and other heavy machinery: prices, availability, specifications and leasing.',
    heading: 'Find a machine by specification and price',
    lead: 'Excavators, loaders, cranes, road and mining machinery. Specifications and price are on the machine page, and the leasing payment is worked out right there.',
    machinesInCatalog: 'Machines in the catalogue',
    sectionsCount: 'Sections',
    leasingFrom: 'Leasing from',
    catalogSections: 'Catalogue sections',
    categoriesFailed: 'Could not load the sections',
    noCategories: 'No sections yet',
    recent: 'Recently added',
    recentFailed: 'Could not load the machines',
    noMachines: 'No machines yet',
    noMachinesHint:
      'We are filling the catalogue — come back later or tell us what you are looking for.',
  },

  contacts: {
    title: 'Contacts',
    metaDescription: 'Address, WhatsApp and request form of TechLine LLP, Almaty.',
    lead: 'We answer during working hours and will find the machine for your job.',
    company: 'Company',
    writeWhatsApp: 'Message on WhatsApp',
    formTitle: 'Request a call',
    formText: 'Leave your number — we will call back and clarify the job.',
  },

  leasingFields: {
    program: 'Programme',
    term: 'Term',
    monthsShort: 'mo.',
    downPayment: 'Down payment',
    rate: 'Rate',
    schedule: 'Schedule',
    annuity: 'Equal instalments',
    differentiated: 'Decreasing',
    boundsRange: (min: string, max: string) => `from ${min} to ${max}`,
    boundsFrom: (min: string) => `from ${min}`,
    machine: 'Machine',
    searching: 'Searching…',
    calcFailed: 'Could not calculate',
    monthlyPayment: 'Monthly payment',
    overpayment: 'Overpayment',
    totalPaid: 'Total paid',
    scheduleToggle: (count: number) => `Payment schedule (${count})`,
    scheduleHide: 'Hide the schedule',
    payment: 'Payment',
    principal: 'Principal',
    interest: 'Interest',
    balance: 'Balance',
  },

  tco: {
    engineHours: (count: number) => (count === 1 ? 'engine hour' : 'engine hours'),
    timesOverTerm: (count: number) => `${count} ${count === 1 ? 'time' : 'times'} over the term`,
    ownershipTerm: 'Ownership term',
    years: 'years',
    hoursPerYear: 'Hours per year',
    engineHoursUnit: 'engine hours',
    diesel: 'Diesel',
    perLitre: '₸ per litre',
    operator: 'Operator',
    perEngineHour: '₸ per engine hour',
    recalculate: 'Recalculate',
    calcFailed: 'Could not calculate',
    costPerHour: 'Cost per engine hour',
    purchase: 'Buying the machine',
    fuel: 'Fuel',
    maintenance: 'Scheduled maintenance',
    residual: 'Residual value',
    totalOverTerm: 'Total over the term',
    missingPrefix: '· not recorded: ',
    noData: 'no data',
    incompleteNote:
      'Only the lines we could calculate are in the total. We do not substitute market averages for values that are not recorded: a calculation with an invented fuel consumption looks like a calculation but is a guess. Ask a manager for what is missing.',
    disclaimer: (rate: string, date: string) =>
      `This is an indicative estimate. It depends on operating conditions: ground, shifts, fuel quality and how the machine is maintained. The machine price is converted to tenge at the rate of ${rate} ₸ on ${date}.`,
  },

  states: {
    loadFailed: 'Could not load the data',
    loading: 'Loading…',
  },

  leasingPage: {
    title: 'Leasing',
    heading: 'Machinery leasing',
    metaDescription:
      'Machinery leasing: terms of the current programmes and a monthly payment calculator with a schedule.',
    lead: 'A machine works and pays for itself while you are still paying for it. Work out the payment before talking to a manager.',
    steps: [
      {
        title: 'Down payment',
        text: 'You pay part of the price up front — usually from 15 %. The larger the down payment, the smaller the monthly payment and the overpayment.',
      },
      {
        title: 'The rest in instalments',
        text: 'The leasing company finances the remaining amount and you repay it on a schedule: equal or decreasing instalments.',
      },
      {
        title: 'The machine is yours',
        text: 'All that time the machine is working for you. After the final payment it becomes your property.',
      },
    ],
    programs: 'Programmes',
    programsFailed: 'Could not load the programmes',
    noPrograms: 'There are no active programmes right now',
    noProgramsHint: 'A manager can confirm the terms — leave a request on the contacts page.',
    downPayment: 'Down payment',
    term: 'Term',
    rate: 'Rate',
    amountCap: 'Amount cap',
    noCap: 'no cap',
    fromValue: (value: string) => `from ${value}`,
    calculator: 'Calculator',
    calculatorNote:
      'The calculation is indicative: it uses today’s exchange rate and does not fix the terms of the deal. Fixed figures come with a formal quotation.',
  },

  service: {
    title: 'Service centre',
    metaTitle: 'Service',
    metaDescription:
      'TechLine service centre in Almaty: diagnostics, warranty and routine repair of machinery, on-site service, pre-delivery inspection.',
    lead: 'Repair and maintenance of machinery in Almaty: our own 3,000 m² centre, original parts, on-site visits.',
    services: [
      {
        title: 'Diagnostics',
        text: 'We find the cause before taking anything apart: bench measurements, hydraulics under load, reading electronic fault codes. The result is a list of work and parts with prices, not “we need to take a look”.',
      },
      {
        title: 'Warranty repair',
        text: 'We service to the manufacturer’s schedule. The warranty stays intact: the work and replaced parts are documented through the manufacturer.',
      },
      {
        title: 'Assembly repair',
        text: 'Bench repair of engines, gearboxes, axles and hydraulics. A rebuilt assembly is tested across the full range of modes before it goes back on the machine.',
      },
      {
        title: 'On-site service',
        text: 'A crew with mobile diagnostic equipment travels to the site. Repair on the spot where the machine cannot be taken off the job or hauled away.',
      },
      {
        title: 'Assembly and pre-delivery inspection',
        text: 'Major-unit assembly, tightening, filling fluids, greasing, testing under load. The machine arrives ready to work, with nothing left to set up.',
      },
    ],
    facilitiesTitle: 'What the centre has',
    facilities: [
      'Hydraulics bay with benches for repair and testing',
      'Machining bay and parts reconditioning',
      'Assembly bay with benches for units',
      'Tool and spare parts store',
      'Machine wash and paint shop',
      'Training room for the customer’s staff',
    ],
    ctaTitle: 'Need a repair or diagnostics?',
    ctaText:
      'Leave your number — an engineer will call back, confirm the model and the nature of the fault and give you a timeline. If the machine cannot be brought in, we will arrange a site visit.',
  },

  about: {
    title: 'About us',
    metaDescription: 'TechLine LLP — machinery supply, spare parts and service across Kazakhstan.',
    lead: 'TechLine LLP — machinery supply, spare parts and service.',
    paragraphs: [
      'We supply construction, road, mining and warehouse machinery: excavators, loaders, cranes, rollers, bulldozers, dump trucks, drilling rigs and attachments. Most of the catalogue is XCMG machinery, with selected models from other manufacturers alongside it.',
      'Our work does not end with delivery. The company runs its own 3,000 m² service centre in Almaty: diagnostics, warranty and routine repair, bench repair of assemblies, on-site visits. We order spare parts and keep them in stock so that downtime is measured in days, not months.',
      'Machinery is not bought with cash on the account alone: we work with leasing. The leasing page lists programme terms and a calculator — the payment can be worked out before talking to a manager.',
    ],
    cities: ['Almaty', 'Astana', 'Shymkent', 'Karaganda', 'Aktobe', 'Aktau'],
    whereWeWork: 'Where we work',
    contactUs: 'Get in touch',
    catalogLink: 'Machinery catalogue',
  },

  spareParts: {
    title: 'Spare parts',
    metaDescription:
      'Spare parts and consumables for heavy machinery: engines, filters, oils, tyres. Selection by model and serial number.',
    lead: 'We select parts by machine model and serial number. Whatever is in stock ships straight away; the rest we order from the factory.',
    groups: [
      {
        title: 'Engines and assemblies',
        text: 'Complete engines, gearboxes, axles, hydraulic pumps and motors, turbochargers, starters.',
      },
      {
        title: 'Filters and consumables',
        text: 'Oil, fuel, air and hydraulic filters, belts, seals and rubber goods.',
      },
      {
        title: 'Oils and technical fluids',
        text: 'Engine and hydraulic oils, transmission fluids, antifreeze, greases.',
      },
      {
        title: 'Undercarriage and tyres',
        text: 'Track chains, rollers, sprockets, shoes, tyres for loaders and quarry machinery.',
      },
    ],
    prepareTitle: 'What to have ready for the call',
    prepare: [
      'Machine model and serial number — they make the selection most accurate.',
      'The part number if you know it: with a catalogue number nothing can go wrong.',
      'What exactly happened — sometimes an assembly repair is needed instead of a part.',
    ],
    requestTitle: 'Spare parts request',
    requestText:
      'Describe what you need and a specialist will find it and quote a price and lead time.',
    messageLabel: 'What you need',
    messagePlaceholder: 'For example: oil filter for XCMG XE215C, serial number 1234567',
  },

  lead: {
    open: 'Request a call',
    dialogTitle: 'Machinery request',
    aboutMachine: (name: string) => `A manager will call back and tell you about “${name}”.`,
    generic: 'A manager will call back and answer your questions.',
    name: 'Name',
    phone: 'Phone',
    city: 'City',
    optional: '— optional',
    nameRequired: 'Tell us how to address you',
    phoneIncomplete: 'Enter the full number: +7 (700) 000-00-00',
    submit: 'Send request',
    consent: 'By pressing the button you agree to the processing of personal data.',
    accepted: 'Request received',
    willCall: (phone: string) => `A manager will call ${phone} during working hours.`,
    noRepeat: 'No need to send it again.',
    charCount: (used: number, max: number) => `${used} of ${max} characters`,
  },

  machinePage: {
    fallbackTitle: 'Machine',
    year: (year: number) => `${year}`,
    delivery: (term: string) => `Delivery in ${term}`,
    photoOf: (index: number, total: number) => `Photo ${index} of ${total}`,
    specs: 'Specifications',
    specFallback: 'Specification',
    noSpecs:
      'Ask a manager for the specifications of this machine — leave a request and we will send them together with delivery terms.',
    askLabel: 'What to clarify',
    askPlaceholder: 'Configuration, delivery times, leasing terms',
    ctaTitle: 'Find the right machine',
    ctaText:
      'We will explain the configuration and delivery times, calculate leasing and suggest an alternative if this machine does not fit.',
    wholeSection: 'Whole section',
    othersIn: (section: string) => `More in ${section}`,
    seeAll: 'See all',
  },

  leasingBlock: {
    title: 'Leasing',
    hint: 'Work out the monthly payment',
    calculate: 'Calculate',
    collapse: 'Collapse',
  },

  tcoBlock: {
    title: 'Cost of ownership',
    perHour: (amount: string) => `${amount} per engine hour`,
    fromPrefix: 'from ',
    incompleteSuffix: ' — not everything counted',
    details: 'Details',
    collapse: 'Collapse',
  },

  compare: {
    add: 'Compare',
    added: 'In comparison',
    addHint: 'Add to comparison',
    removeHint: 'Remove from comparison',
    fullHint: (max: number) => `${max} machines already selected — no more fit on screen`,
    trayCount: (count: number, max: number) => `Selected for comparison: ${count} of ${max}`,
    clear: 'Clear',
    needSecond: 'Add a second one',
  },

  section: {
    home: 'Home',
    subcategories: 'Subcategories',
    notFoundTitle: 'Section not found',
    catalogTitle: 'Catalogue',
    metaDescription: (name: string) =>
      `${name} — Techline machinery catalogue: prices, availability and specifications.`,
    emptyByFilters: 'Nothing matches the selected filters',
    emptyByFiltersHint: 'Try widening the ranges or dropping some conditions.',
    emptySection: 'This section is empty for now',
    resetFilters: 'Reset filters',
  },

  filters: {
    title: 'Filters',
    reset: 'Reset',
    liveHint: 'Results update as you choose',
    price: 'Price',
    availability: 'Availability',
    condition: 'Condition',
    productionYear: 'Year built',
    search: 'Search',
    searchPlaceholder: 'Name, brand or model',
    from: 'from',
    to: 'to',
    rangeHint: (min: string, max: string) => `In this section: ${min} to ${max}`,
    singleValueHint: (value: string) => `In this section: ${value} only`,
    fieldFrom: (label: string) => `${label}: from`,
    fieldTo: (label: string) => `${label}: to`,
    showMore: (count: number) => `More filters (${count})`,
    showLess: 'Fewer filters',
    unavailable: 'Specification filters are unavailable',
  },

  sort: {
    label: 'Sort',
    published_desc: 'Newest first',
    price_asc: 'Cheapest first',
    price_desc: 'Most expensive first',
  },

  grid: {
    ratePrefix: (date: string) => `Prices in tenge — at the rate on ${date}`,
    ratePrefixBoth: (date: string) => `Prices in tenge and dollars — at the rates on ${date}`,
    ratePrefixSplit: (kztDate: string, usdDate: string) =>
      `Prices in tenge — at the rate on ${kztDate}, in dollars — on ${usdDate}`,
    loadMore: 'Show more',
  },

  catalog: {
    found: (total: number, shown: number) => {
      const found = `${total} ${total === 1 ? 'machine' : 'machines'} found`

      return total > shown ? `${found}, showing the first ${shown}` : found
    },
  },

  search: {
    nothingFound: 'Nothing found',
    sectionsOnly: 'No machines match that name, but there is a matching section',
  },

  units: {
    months: (count: number) => `${count} ${count === 1 ? 'month' : 'months'}`,
  },

  theme: {
    toDark: 'Switch to dark theme',
    toLight: 'Switch to light theme',
    dark: 'Dark theme',
    light: 'Light theme',
  },
}
