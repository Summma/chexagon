import argparse
import json
import torch

from nnue import NNUEProbability, NNUECentipawn


def export_model(model_path, output_path, model_type="probability"):
    """Export a PyTorch model to JSON format for Go inference."""
    
    # Load model
    if model_type == "probability":
        model = NNUEProbability()
    else:
        model = NNUECentipawn()
    
    model.load_state_dict(torch.load(model_path, map_location="cpu", weights_only=True))
    model.eval()
    
    # Extract weights
    state_dict = model.state_dict()
    
    weights = {}
    for name, tensor in state_dict.items():
        weights[name] = {
            "shape": list(tensor.shape),
            "data": tensor.numpy().flatten().tolist()
        }
    
    # Add metadata
    export_data = {
        "model_type": model_type,
        "weights": weights
    }
    
    # Save to JSON
    with open(output_path, "w") as f:
        json.dump(export_data, f)
    
    print(f"Exported {model_path} to {output_path}")
    print(f"Model type: {model_type}")
    print("Layers:")
    for name, tensor in state_dict.items():
        print(f"  {name}: {list(tensor.shape)}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Export PyTorch NNUE model to JSON")
    parser.add_argument("model", help="Path to .pt model file")
    parser.add_argument("-o", "--output", default=None, help="Output JSON path (default: model.json)")
    parser.add_argument("--type", choices=["probability", "centipawn"], default="probability",
                        help="Model type")
    args = parser.parse_args()
    
    output_path = args.output or args.model.replace(".pt", ".json")
    export_model(args.model, output_path, args.type)
