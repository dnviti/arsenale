const demoDb = db.getSiblingDB('arsenale_demo');

if (!demoDb.getUser('demo_mongo_user')) {
  demoDb.createUser({
    user: 'demo_mongo_user',
    pwd: 'DemoMongoPass123!',
    roles: [{ role: 'readWrite', db: 'arsenale_demo' }],
  });
}

const collectionNames = [
  'demo_payments',
  'demo_invoices',
  'demo_purchase_order_items',
  'demo_purchase_orders',
  'demo_order_items',
  'demo_orders',
  'demo_inventory',
  'demo_customer_addresses',
  'demo_products',
  'demo_product_categories',
  'demo_suppliers',
  'demo_warehouses',
  'demo_employees',
  'demo_customers',
];

for (const name of collectionNames) {
  if (demoDb.getCollectionNames().includes(name)) {
    demoDb[name].drop();
  }
}

function addDays(value, days) {
  return new Date(value.getTime() + days * 24 * 60 * 60 * 1000);
}

function regionFor(index) {
  switch ((index - 1) % 5) {
    case 0: return 'na';
    case 1: return 'emea';
    case 2: return 'apac';
    case 3: return 'latam';
    default: return 'mea';
  }
}

function countryFor(region) {
  switch (region) {
    case 'na': return 'United States';
    case 'emea': return 'Germany';
    case 'apac': return 'Singapore';
    case 'latam': return 'Brazil';
    default: return 'South Africa';
  }
}

function currencyForCustomer(customerId) {
  switch ((customerId - 1) % 5) {
    case 0: return 'USD';
    case 1: return 'EUR';
    case 2: return 'SGD';
    case 3: return 'BRL';
    default: return 'ZAR';
  }
}

function productCategoryFor(productId) {
  return Math.floor((productId - 1) / 9) + 1;
}

function productSequenceFor(productId) {
  return ((productId - 1) % 9) + 1;
}

function unitCostFor(productId) {
  const categoryId = productCategoryFor(productId);
  const seq = productSequenceFor(productId);
  return Number((18 + categoryId * 9 + seq * 2.5 + (productId % 3) * 1.1).toFixed(2));
}

function unitPriceFor(productId) {
  const categoryId = productCategoryFor(productId);
  const multiplier = categoryId % 3 === 0 ? 1.6 : categoryId % 3 === 1 ? 1.45 : 1.55;
  return Number((unitCostFor(productId) * multiplier).toFixed(2));
}

const customers = [];
const customerAddresses = [];
const suppliers = [];
const employees = [];
const categories = [
  { _id: 1, category_code: 'NET', name: 'Networking', margin_target: 0.32, active: true },
  { _id: 2, category_code: 'END', name: 'Endpoints', margin_target: 0.28, active: true },
  { _id: 3, category_code: 'COL', name: 'Collaboration', margin_target: 0.35, active: true },
  { _id: 4, category_code: 'SEC', name: 'Security', margin_target: 0.38, active: true },
  { _id: 5, category_code: 'INF', name: 'Infrastructure', margin_target: 0.30, active: true },
  { _id: 6, category_code: 'IND', name: 'Industrial', margin_target: 0.33, active: true },
  { _id: 7, category_code: 'OFF', name: 'Office', margin_target: 0.25, active: true },
  { _id: 8, category_code: 'SVC', name: 'Services', margin_target: 0.45, active: true },
];
const warehouses = [
  { _id: 1, warehouse_code: 'WH-NA1', name: 'Atlanta Distribution Center', region: 'na', country: 'United States', capacity_units: 18000, active: true },
  { _id: 2, warehouse_code: 'WH-EMEA1', name: 'Rotterdam Hub', region: 'emea', country: 'Netherlands', capacity_units: 15000, active: true },
  { _id: 3, warehouse_code: 'WH-APAC1', name: 'Singapore Fulfillment Hub', region: 'apac', country: 'Singapore', capacity_units: 14000, active: true },
  { _id: 4, warehouse_code: 'WH-LATAM1', name: 'Sao Paulo Distribution Center', region: 'latam', country: 'Brazil', capacity_units: 12000, active: true },
  { _id: 5, warehouse_code: 'WH-MEA1', name: 'Johannesburg Regional Hub', region: 'mea', country: 'South Africa', capacity_units: 10000, active: true },
];
const products = [];
const inventory = [];
const orders = [];
const orderItems = [];
const purchaseOrders = [];
const purchaseOrderItems = [];
const invoices = [];
const payments = [];

for (let i = 1; i <= 15; i += 1) {
  const region = regionFor(i);
  suppliers.push({
    _id: i,
    supplier_code: `SUP-${String(i).padStart(3, '0')}`,
    supplier_name: `Global Supply Group ${String(i).padStart(3, '0')}`,
    contact_name: `Vendor Contact ${String(i).padStart(3, '0')}`,
    email: `vendor${String(i).padStart(3, '0')}@supplier${String(i).padStart(3, '0')}.example.dev`,
    phone: `+1-800-55${String(i).padStart(4, '0')}`,
    region,
    country: countryFor(region),
    payment_terms_days: [30, 45, 60, 15][i % 4],
    active: i % 11 !== 0,
    created_at: addDays(new Date('2024-01-01T08:00:00Z'), i * 6),
  });
}

for (let i = 1; i <= 18; i += 1) {
  const region = regionFor(i);
  const department = ['Sales', 'Operations', 'Finance', 'Procurement', 'Customer Success'][(i - 1) % 5];
  const title = {
    Sales: 'Account Executive',
    Operations: 'Operations Manager',
    Finance: 'Finance Analyst',
    Procurement: 'Procurement Lead',
    'Customer Success': 'Success Manager',
  }[department];
  employees.push({
    _id: i,
    employee_code: `EMP-${String(i).padStart(3, '0')}`,
    full_name: `Employee ${String(i).padStart(3, '0')}`,
    department,
    title,
    email: `employee${String(i).padStart(3, '0')}@example.dev`,
    region,
    hired_at: addDays(new Date('2021-01-01T09:00:00Z'), i * 21),
    active: i % 13 !== 0,
  });
}

for (let i = 1; i <= 60; i += 1) {
  const region = regionFor(i);
  const country = countryFor(region);
  const segment = ['enterprise', 'mid_market', 'public_sector', 'smb'][(i - 1) % 4];
  const prefix = ['Crescent', 'Blue Harbor', 'Summit', 'Atlas', 'Riverbank', 'Northwind'][i % 6];
  const sector = ['Energy', 'Manufacturing', 'Logistics', 'Healthcare', 'Retail'][i % 5];
  customers.push({
    _id: i,
    customer_code: `CUST-${String(i).padStart(3, '0')}`,
    email: `contact${String(i).padStart(3, '0')}@customer${String(i).padStart(3, '0')}.example.dev`,
    full_name: `Contact ${String(i).padStart(3, '0')}`,
    company_name: `${prefix} ${sector} ${String(i).padStart(3, '0')}`,
    segment,
    region,
    country,
    phone: `+1-555-2${String(i).padStart(4, '0')}`,
    active: i % 17 !== 0,
    credit_limit: 5000 + i * 750,
    tax_id: `TIN-${String(i).padStart(6, '0')}`,
    created_at: addDays(new Date('2024-06-01T10:00:00Z'), i * 2),
    updated_at: addDays(new Date('2024-06-01T10:00:00Z'), i * 2 + ((i % 7) + 1)),
  });
  customerAddresses.push(
    {
      _id: (i - 1) * 2 + 1,
      customer_id: i,
      address_type: 'billing',
      line1: `${100 + i} Commerce Avenue`,
      line2: `Suite ${200 + (i % 25)}`,
      city: { na: 'Atlanta', emea: 'Berlin', apac: 'Singapore', latam: 'Sao Paulo', mea: 'Johannesburg' }[region],
      state_region: { na: 'GA', emea: 'BE', apac: 'SG', latam: 'SP', mea: 'GP' }[region],
      postal_code: `${String(30 + (i % 60)).padStart(2, '0')}${String(i).padStart(3, '0')}`,
      country,
      is_primary: true,
    },
    {
      _id: (i - 1) * 2 + 2,
      customer_id: i,
      address_type: 'shipping',
      line1: `${500 + i} Logistics Park`,
      line2: null,
      city: { na: 'Dallas', emea: 'Hamburg', apac: 'Jurong', latam: 'Campinas', mea: 'Cape Town' }[region],
      state_region: { na: 'TX', emea: 'HH', apac: 'SG', latam: 'SP', mea: 'WC' }[region],
      postal_code: `${String(70 + (i % 40)).padStart(2, '0')}${String(i).padStart(3, '0')}`,
      country,
      is_primary: false,
    },
  );
}

for (let i = 1; i <= 72; i += 1) {
  const categoryId = productCategoryFor(i);
  const seq = productSequenceFor(i);
  products.push({
    _id: i,
    sku: `SKU-${String(i).padStart(4, '0')}`,
    category_id: categoryId,
    supplier_id: ((i - 1) % 15) + 1,
    name: `${['Network', 'Endpoint', 'Collaboration', 'Security', 'Infrastructure', 'Industrial', 'Office', 'Service'][categoryId - 1]} Item ${String(seq).padStart(2, '0')}`,
    description: `Catalog product ${String(i).padStart(4, '0')} in category ${categoryId}`,
    unit_cost: unitCostFor(i),
    unit_price: unitPriceFor(i),
    currency: 'USD',
    active: i % 19 !== 0,
    created_at: addDays(new Date('2024-03-01T07:00:00Z'), i),
  });
}

for (const warehouse of warehouses) {
  for (const product of products) {
    inventory.push({
      _id: inventory.length + 1,
      warehouse_id: warehouse._id,
      product_id: product._id,
      on_hand_qty: 25 + ((product._id * warehouse._id * 3) % 180),
      reserved_qty: (product._id + warehouse._id) % 14,
      reorder_point: 18 + (product._id % 12),
      reorder_qty: 30 + warehouse._id * 15,
      last_counted_at: addDays(new Date('2025-02-01T08:00:00Z'), (product._id + warehouse._id) % 25),
    });
  }
}

for (let i = 1; i <= 180; i += 1) {
  const customerId = ((i - 1) % 60) + 1;
  const status = ['paid', 'confirmed', 'picking', 'shipped', 'invoiced', 'draft'][i % 6];
  const orderedAt = addDays(new Date('2025-01-01T09:30:00Z'), i);
  const order = {
    _id: i,
    order_number: `SO-2025-${String(i).padStart(5, '0')}`,
    customer_id: customerId,
    sales_rep_id: ((i - 1) % 18) + 1,
    status,
    currency: currencyForCustomer(customerId),
    subtotal: 0,
    discount_total: 0,
    tax_total: 0,
    order_total: 0,
    ordered_at: orderedAt,
    requested_ship_at: addDays(orderedAt, 7),
    shipped_at: ['shipped', 'invoiced', 'paid'].includes(status) ? addDays(orderedAt, 4 + (i % 5)) : null,
    channel: ['web', 'partner', 'inside_sales', 'renewal'][i % 4],
    payment_terms_days: [15, 30, 45, 60][i % 4],
    notes: `Seeded ERP sales order ${String(i).padStart(5, '0')}`,
  };
  orders.push(order);

  let grossTotal = 0;
  let discountTotal = 0;
  let netTotal = 0;
  for (let lineNo = 1; lineNo <= 3; lineNo += 1) {
    const productId = ((i * 7 + lineNo * 11 - 1) % 72) + 1;
    const quantity = ((i + lineNo) % 5) + 1;
    const discountPct = i % 12 === 0 ? 0.10 : (lineNo === 3 && i % 5 === 0 ? 0.05 : 0.0);
    const unitPrice = unitPriceFor(productId);
    const lineTotal = Number((quantity * unitPrice * (1 - discountPct)).toFixed(2));
    orderItems.push({
      _id: (i - 1) * 3 + lineNo,
      order_id: i,
      line_number: lineNo,
      product_id: productId,
      quantity,
      unit_price: unitPrice,
      discount_pct: discountPct,
      line_total: lineTotal,
    });
    grossTotal += quantity * unitPrice;
    discountTotal += quantity * unitPrice * discountPct;
    netTotal += lineTotal;
  }
  order.subtotal = Number(grossTotal.toFixed(2));
  order.discount_total = Number(discountTotal.toFixed(2));
  order.tax_total = Number((netTotal * 0.08).toFixed(2));
  order.order_total = Number((netTotal + order.tax_total).toFixed(2));
}

for (let i = 1; i <= 72; i += 1) {
  const status = ['closed', 'sent', 'confirmed', 'received', 'planned'][i % 5];
  const orderedAt = addDays(new Date('2024-11-01T08:15:00Z'), i * 3);
  const po = {
    _id: i,
    po_number: `PO-2025-${String(i).padStart(5, '0')}`,
    supplier_id: ((i - 1) % 15) + 1,
    buyer_id: ((i - 1) % 18) + 1,
    status,
    currency: 'USD',
    subtotal: 0,
    tax_total: 0,
    total_amount: 0,
    ordered_at: orderedAt,
    expected_at: addDays(orderedAt, 14 + (i % 7)),
    received_at: ['received', 'closed'].includes(status) ? addDays(orderedAt, 12 + (i % 5)) : null,
  };
  purchaseOrders.push(po);

  let subtotal = 0;
  for (let lineNo = 1; lineNo <= 3; lineNo += 1) {
    const productId = ((i * 5 + lineNo * 13 - 1) % 72) + 1;
    const quantity = 10 + ((i + lineNo) % 25);
    const unitCost = unitCostFor(productId);
    const lineTotal = Number((quantity * unitCost).toFixed(2));
    purchaseOrderItems.push({
      _id: (i - 1) * 3 + lineNo,
      purchase_order_id: i,
      line_number: lineNo,
      product_id: productId,
      quantity,
      unit_cost: unitCost,
      line_total: lineTotal,
    });
    subtotal += lineTotal;
  }
  po.subtotal = Number(subtotal.toFixed(2));
  po.tax_total = Number((subtotal * 0.05).toFixed(2));
  po.total_amount = Number((subtotal + po.tax_total).toFixed(2));
}

for (const order of orders) {
  const invoice = {
    _id: order._id,
    invoice_number: `INV-2025-${String(order._id).padStart(5, '0')}`,
    order_id: order._id,
    customer_id: order.customer_id,
    status: order.status === 'draft' ? 'draft' : (order.status === 'paid' ? 'paid' : 'open'),
    issued_at: addDays(order.ordered_at, 1),
    due_at: addDays(order.ordered_at, order.payment_terms_days),
    currency: order.currency,
    subtotal: Number((order.subtotal - order.discount_total).toFixed(2)),
    tax_total: order.tax_total,
    total_amount: order.order_total,
    balance_due: order.order_total,
  };
  invoices.push(invoice);

  if (['shipped', 'invoiced', 'paid'].includes(order.status)) {
    const amount = order.status === 'shipped'
      ? Number((order.order_total * 0.25).toFixed(2))
      : order.status === 'invoiced'
        ? Number((order.order_total * 0.60).toFixed(2))
        : order.order_total;
    payments.push({
      _id: order._id,
      invoice_id: order._id,
      payment_reference: `PAY-2025-${String(order._id).padStart(5, '0')}`,
      payment_method: ['wire', 'card', 'ach', 'bank_transfer'][order._id % 4],
      amount,
      currency: order.currency,
      paid_at: addDays(order.ordered_at, 5 + (order._id % 12)),
      status: order.status === 'paid' ? 'settled' : 'posted',
      processor: ['stripe', 'adyen', 'bank'][order._id % 3],
    });
  }
}

const paymentsByInvoice = new Map();
for (const payment of payments) {
  paymentsByInvoice.set(payment.invoice_id, Number(((paymentsByInvoice.get(payment.invoice_id) || 0) + payment.amount).toFixed(2)));
}
for (const invoice of invoices) {
  const paid = paymentsByInvoice.get(invoice._id) || 0;
  invoice.balance_due = Number((invoice.total_amount - paid).toFixed(2));
  invoice.status = invoice.balance_due <= 0
    ? 'paid'
    : paid > 0
      ? 'partial'
      : invoice.status === 'draft'
        ? 'draft'
        : 'open';
}

demoDb.demo_customers.insertMany(customers);
demoDb.demo_customer_addresses.insertMany(customerAddresses);
demoDb.demo_suppliers.insertMany(suppliers);
demoDb.demo_employees.insertMany(employees);
demoDb.demo_product_categories.insertMany(categories);
demoDb.demo_products.insertMany(products);
demoDb.demo_warehouses.insertMany(warehouses);
demoDb.demo_inventory.insertMany(inventory);
demoDb.demo_orders.insertMany(orders);
demoDb.demo_order_items.insertMany(orderItems);
demoDb.demo_purchase_orders.insertMany(purchaseOrders);
demoDb.demo_purchase_order_items.insertMany(purchaseOrderItems);
demoDb.demo_invoices.insertMany(invoices);
demoDb.demo_payments.insertMany(payments);

demoDb.demo_customers.createIndex({ customer_code: 1 }, { unique: true });
demoDb.demo_customers.createIndex({ email: 1 }, { unique: true });
demoDb.demo_customer_addresses.createIndex({ customer_id: 1 });
demoDb.demo_suppliers.createIndex({ supplier_code: 1 }, { unique: true });
demoDb.demo_employees.createIndex({ employee_code: 1 }, { unique: true });
demoDb.demo_products.createIndex({ sku: 1 }, { unique: true });
demoDb.demo_products.createIndex({ category_id: 1 });
demoDb.demo_products.createIndex({ supplier_id: 1 });
demoDb.demo_inventory.createIndex({ warehouse_id: 1, product_id: 1 }, { unique: true });
demoDb.demo_orders.createIndex({ order_number: 1 }, { unique: true });
demoDb.demo_orders.createIndex({ customer_id: 1, status: 1 });
demoDb.demo_order_items.createIndex({ order_id: 1, line_number: 1 }, { unique: true });
demoDb.demo_purchase_orders.createIndex({ po_number: 1 }, { unique: true });
demoDb.demo_purchase_order_items.createIndex({ purchase_order_id: 1, line_number: 1 }, { unique: true });
demoDb.demo_invoices.createIndex({ invoice_number: 1 }, { unique: true });
demoDb.demo_payments.createIndex({ payment_reference: 1 }, { unique: true });
