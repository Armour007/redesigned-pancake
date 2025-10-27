<?php
/**
 * Plugin Name: Aura Verify
 * Description: Gate WooCommerce actions via Aura real-time verification.
 * Version: 0.1.0
 * Author: Aura
 */

if ( ! defined( 'ABSPATH' ) ) { exit; }

class Aura_Verify_Plugin {
    const OPTION_API_KEY = 'aura_verify_api_key';
    const OPTION_AGENT_ID = 'aura_verify_agent_id';
    const OPTION_API_BASE = 'aura_verify_api_base'; // default http://localhost:8081

    public function __construct() {
        add_action('admin_menu', [$this, 'add_settings_page']);
        add_action('admin_init', [$this, 'register_settings']);

        // Example: verify on order status change to processing/completed
        add_action('woocommerce_order_status_changed', [$this, 'on_order_status_changed'], 10, 4);
    }

    public function add_settings_page() {
        add_menu_page(
            'Aura Verify', 'Aura Verify', 'manage_options', 'aura-verify', [$this, 'render_settings_page'], 'dashicons-shield', 80
        );
    }

    public function register_settings() {
        register_setting('aura_verify_settings', self::OPTION_API_KEY);
        register_setting('aura_verify_settings', self::OPTION_AGENT_ID);
        register_setting('aura_verify_settings', self::OPTION_API_BASE);
    }

    public function render_settings_page() {
        ?>
        <div class="wrap">
            <h1>Aura Verify</h1>
            <form method="post" action="options.php">
                <?php settings_fields('aura_verify_settings'); ?>
                <?php do_settings_sections('aura_verify_settings'); ?>
                <table class="form-table">
                    <tr>
                        <th scope="row"><label for="api_key">API Key</label></th>
                        <td><input type="text" name="<?php echo self::OPTION_API_KEY; ?>" value="<?php echo esc_attr(get_option(self::OPTION_API_KEY)); ?>" class="regular-text" /></td>
                    </tr>
                    <tr>
                        <th scope="row"><label for="agent_id">Agent ID</label></th>
                        <td><input type="text" name="<?php echo self::OPTION_AGENT_ID; ?>" value="<?php echo esc_attr(get_option(self::OPTION_AGENT_ID)); ?>" class="regular-text" /></td>
                    </tr>
                    <tr>
                        <th scope="row"><label for="api_base">API Base</label></th>
                        <td><input type="text" name="<?php echo self::OPTION_API_BASE; ?>" value="<?php echo esc_attr(get_option(self::OPTION_API_BASE, 'http://localhost:8081')); ?>" class="regular-text" /></td>
                    </tr>
                </table>
                <?php submit_button(); ?>
            </form>
        </div>
        <?php
    }

    public function on_order_status_changed($order_id, $old_status, $new_status, $order) {
        // Only verify on transitions to processing or completed
        if (!in_array($new_status, ['processing','completed'])) { return; }

        $api_key = get_option(self::OPTION_API_KEY);
        $agent_id = get_option(self::OPTION_AGENT_ID);
        $api_base = get_option(self::OPTION_API_BASE, 'http://localhost:8081');
        if (empty($api_key) || empty($agent_id)) { return; }

        $context = [
            'action' => 'woocommerce.order.update',
            'env' => 'prod', // TODO: make configurable
            'order' => [
                'id' => $order_id,
                'total' => $order->get_total(),
                'currency' => $order->get_currency(),
                'status' => $new_status,
            ],
        ];
        $body = [
            'agent_id' => $agent_id,
            'request_context' => $context,
        ];

        $args = [
            'headers' => [
                'Content-Type' => 'application/json',
                'X-API-Key' => $api_key,
                'AURA-Version' => '2025-10-01',
            ],
            'body' => wp_json_encode($body),
            'timeout' => 5,
        ];
        $resp = wp_remote_post(trailingslashit($api_base) . 'v1/verify', $args);
        if (is_wp_error($resp)) {
            error_log('[Aura Verify] request error: ' . $resp->get_error_message());
            return;
        }
        $code = wp_remote_retrieve_response_code($resp);
        $data = json_decode(wp_remote_retrieve_body($resp), true);
        if ($code !== 200 || !isset($data['decision'])) {
            error_log('[Aura Verify] unexpected response: ' . $code . ' body=' . wp_remote_retrieve_body($resp));
            return;
        }
        if ($data['decision'] !== 'ALLOWED') {
            // Optionally revert status or add order note
            $order->add_order_note('Aura denied action: ' . ($data['reason'] ?? '')); 
            // TODO: make policy configurable (block vs. warn)
        } else {
            $order->add_order_note('Aura allowed action');
        }
    }
}

new Aura_Verify_Plugin();
