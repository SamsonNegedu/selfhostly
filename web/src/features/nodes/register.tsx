import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useRegisterNode } from '../../shared/services/api';
import { Button } from '../../shared/components/ui/button';
import { Card } from '../../shared/components/ui/card';
import { Input } from '../../shared/components/ui/input';
import { ArrowLeft, HelpCircle } from 'lucide-react';
import type { RegisterNodeRequest } from '../../shared/types/api';

export function RegisterNodePage() {
  const navigate = useNavigate();
  const registerMutation = useRegisterNode();
  const [formData, setFormData] = useState<RegisterNodeRequest>({
    name: '',
    api_endpoint: '',
    api_key: '',
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await registerMutation.mutateAsync(formData);
      navigate('/nodes');
    } catch (error) {
      console.error('Failed to register node:', error);
    }
  };

  const isValid = formData.name && formData.api_endpoint && formData.api_key;

  return (
    <div className="fade-in space-y-6">
      {/* Header */}
      <div>
        <Button
          variant="ghost"
          onClick={() => navigate('/nodes')}
          className="mb-4"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Nodes
        </Button>
        <h1 className="text-3xl font-bold">Register New Node</h1>
        <p className="text-muted-foreground mt-2">
          Add a secondary node to your cluster for distributed app deployment
        </p>
      </div>

      {/* Instructions */}
      <Card className="p-6 bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-800">
        <div className="flex gap-3">
          <HelpCircle className="h-5 w-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
          <div>
            <h3 className="font-medium mb-2">Before registering a node</h3>
            <ol className="text-sm text-muted-foreground space-y-1 list-decimal list-inside">
              <li>Install Selfhostly on the secondary machine</li>
              <li>Configure with <code className="bg-blue-100 dark:bg-blue-900 px-1 rounded">NODE_IS_PRIMARY=false</code></li>
              <li>Set <code className="bg-blue-100 dark:bg-blue-900 px-1 rounded">PRIMARY_NODE_URL</code> to this node's URL</li>
              <li>Start the secondary node and copy the registration details from startup logs</li>
            </ol>
            <p className="text-xs text-muted-foreground mt-3 pl-5">
              The startup logs will display the node name, API endpoint, and API key needed for registration.
            </p>
          </div>
        </div>
      </Card>

      {/* Registration Form */}
      <Card className="p-6">
        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label className="block text-sm font-medium mb-2">
              Node Name <span className="text-red-500">*</span>
            </label>
            <Input
              placeholder="node2"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
            />
            <p className="text-xs text-muted-foreground mt-1">
              Unique identifier for this node (e.g., rpi-node-2, us-west-1)
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">
              API Endpoint <span className="text-red-500">*</span>
            </label>
            <Input
              type="url"
              placeholder="https://node2.example.com or http://192.168.1.100:8080"
              value={formData.api_endpoint}
              onChange={(e) => setFormData({ ...formData, api_endpoint: e.target.value })}
              required
            />
            <p className="text-xs text-muted-foreground mt-1">
              Full URL where this node's API is accessible from this primary node
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">
              API Key <span className="text-red-500">*</span>
            </label>
            <Input
              type="password"
              placeholder="Enter the node's API key"
              value={formData.api_key}
              onChange={(e) => setFormData({ ...formData, api_key: e.target.value })}
              required
            />
            <p className="text-xs text-muted-foreground mt-1">
              Copy this from the secondary node's startup logs (automatically generated if not set)
            </p>
          </div>

          {/* Error Display */}
          {registerMutation.isError && (
            <div className="p-4 bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded-lg">
              <p className="text-sm text-red-800 dark:text-red-200">
                Failed to register node: {registerMutation.error instanceof Error ? registerMutation.error.message : 'Unknown error'}
              </p>
            </div>
          )}

          {/* Actions */}
          <div className="flex gap-3 justify-end pt-4">
            <Button
              type="button"
              variant="ghost"
              onClick={() => navigate('/nodes')}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!isValid || registerMutation.isPending}
            >
              {registerMutation.isPending ? 'Registering...' : 'Register Node'}
            </Button>
          </div>
        </form>
      </Card>

      {/* Additional Help */}
      <Card className="p-6">
        <h3 className="font-medium mb-3">Need help?</h3>
        <ul className="text-sm text-muted-foreground space-y-2 list-disc list-inside">
          <li>The node must be reachable from this primary node</li>
          <li>Ensure firewall rules allow traffic on the configured port (default: 8080)</li>
          <li>All registration details are displayed in a formatted box in the startup logs</li>
          <li>For HTTPS endpoints, ensure valid SSL certificates are configured</li>
          <li>Use the full URL including protocol (http:// or https://) for the API endpoint</li>
        </ul>
      </Card>
    </div>
  );
}

export default RegisterNodePage;
