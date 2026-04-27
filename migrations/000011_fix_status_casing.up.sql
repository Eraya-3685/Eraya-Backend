-- +migrate Up
UPDATE orders SET order_status = 'Pending' WHERE order_status = 'pending';
UPDATE orders SET payment_status = 'Pending' WHERE payment_status = 'pending';
