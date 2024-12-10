import pandas as pd
import json
import sys
from statsmodels.tsa.stattools import grangercausalitytests

def granger_causality_analysis(max_lag=3):
    input_file = "/home/vasti/Hobby/skripsi/tmp/nasa_data.csv"
    output_file = "/home/vasti/Hobby/skripsi/granger.json"
    try:
        # Load the dataset
        data = pd.read_csv(input_file)
        
        # Extract relevant columns (excluding YEAR and DOY)
        variables = data.columns[2:]
        
        results = {}
        
        # Perform Granger causality test for all pairs of variables
        for x in variables:
            results[x] = {}
            for y in variables:
                if x != y:
                    try:
                        test_result = grangercausalitytests(
                            data[[x, y]], max_lag
                        )
                        p_values = [round(test[0]['ssr_chi2test'][1], 4) for _, test in test_result.items()]
                        results[x][y] = {"min_p_value": min(p_values), "p_values": p_values}
                    except Exception as e:
                        results[x][y] = {"error": str(e)}
        
        # Save results to a JSON file
        with open(output_file, 'w') as f:
            json.dump(results, f, indent=4)
        
        print(f"Analysis complete. Results saved to {output_file}")
    
    except Exception as e:
        print(f"An error occurred: {e}")

if __name__ == "__main__":
    granger_causality_analysis()
